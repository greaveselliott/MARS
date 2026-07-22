/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-018-goreleaser-distribution.md
*/
package selfupdate

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

const (
	marsReleaseRepository       = "greaveselliott/MARS"
	marsReleaseWorkflow         = "release.yml"
	sigstoreActionsIssuer       = "https://token.actions.githubusercontent.com"
	pinnedTrustedRootSHA256     = "6494e21ea73fa7ee769f85f57d5a3e6a08725eae1e38c755fc3517c9e6bc0b66"
	maxSignedChecksumsBytes     = 16 << 10
	maxSigstoreBundleBytes      = 1 << 20
	expectedSignedChecksumCount = 8
)

var (
	exactReleaseTagPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	exactCommitPattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	exactChecksumPattern   = regexp.MustCompile(`^([0-9a-f]{64})  ([A-Za-z0-9._-]+)$`)

	// These errors deliberately contain no upstream parser, certificate, URL,
	// checksum, or candidate value. Callers may safely report the fixed stage.
	ErrSignedReleaseIdentity  = errors.New("signed release verification: require an exact vMAJOR.MINOR.PATCH tag and 40-character lowercase commit")
	ErrSignedReleaseEvidence  = errors.New("signed release verification: authenticity evidence is invalid; do not install or replace the current binary")
	ErrSignedReleaseChecksums = errors.New("signed release verification: checksum contract is invalid; do not install or replace the current binary")
)

//go:embed trusted_root.json
var pinnedTrustedRootJSON []byte

// SignedChecksums contains the authenticated artifact digests for one MARS
// release. Its map is private so verification results cannot be confused with
// caller-owned input maps.
type SignedChecksums struct {
	digests map[string][sha256.Size]byte
}

// Digest returns one authenticated SHA-256 digest.
func (s SignedChecksums) Digest(name string) ([sha256.Size]byte, bool) {
	digest, ok := s.digests[name]
	return digest, ok
}

// Len returns the number of authenticated checksum records.
func (s SignedChecksums) Len() int {
	return len(s.digests)
}

// VerifyMARSSignedChecksums verifies the Sigstore evidence over the exact raw
// checksums bytes, then enforces the canonical eight-artifact MARS contract.
// It performs no network or filesystem I/O and does not inspect archives.
func VerifyMARSSignedChecksums(checksums, bundleJSON []byte, tag, fullCommit string) (SignedChecksums, error) {
	policy, err := newSigstoreReleasePolicy(marsReleaseRepository, marsReleaseWorkflow, tag, fullCommit)
	if err != nil {
		return SignedChecksums{}, ErrSignedReleaseIdentity
	}
	if len(checksums) == 0 || len(checksums) > maxSignedChecksumsBytes || len(bundleJSON) == 0 || len(bundleJSON) > maxSigstoreBundleBytes {
		return SignedChecksums{}, ErrSignedReleaseEvidence
	}
	// Both verification phases consume the same bounded snapshots; later caller
	// reuse cannot change the bytes authenticated by this invocation.
	checksumsSnapshot := bytes.Clone(checksums)
	bundleSnapshot := bytes.Clone(bundleJSON)
	if err := verifySigstoreChecksumsEvidence(checksumsSnapshot, bundleSnapshot, pinnedTrustedRootJSON, pinnedTrustedRootSHA256, policy); err != nil {
		return SignedChecksums{}, ErrSignedReleaseEvidence
	}
	digests, err := parseCanonicalMARSSignedChecksums(checksumsSnapshot, strings.TrimPrefix(tag, "v"))
	if err != nil {
		return SignedChecksums{}, ErrSignedReleaseChecksums
	}
	return SignedChecksums{digests: digests}, nil
}

type sigstoreReleasePolicy struct {
	identity verify.CertificateIdentity
}

func newSigstoreReleasePolicy(repository, workflow, tag, fullCommit string) (sigstoreReleasePolicy, error) {
	if !exactReleaseTagPattern.MatchString(tag) || !exactCommitPattern.MatchString(fullCommit) {
		return sigstoreReleasePolicy{}, ErrSignedReleaseIdentity
	}
	if repository == "" || workflow == "" || strings.ContainsAny(repository+workflow, "\r\n\x00") {
		return sigstoreReleasePolicy{}, ErrSignedReleaseIdentity
	}
	ref := "refs/tags/" + tag
	repositoryURI := "https://github.com/" + repository
	workflowURI := repositoryURI + "/.github/workflows/" + workflow + "@" + ref
	san, err := verify.NewSANMatcher(workflowURI, "")
	if err != nil {
		return sigstoreReleasePolicy{}, ErrSignedReleaseIdentity
	}
	issuer, err := verify.NewIssuerMatcher(sigstoreActionsIssuer, "")
	if err != nil {
		return sigstoreReleasePolicy{}, ErrSignedReleaseIdentity
	}
	identity, err := verify.NewCertificateIdentity(san, issuer, certificate.Extensions{
		GithubWorkflowSHA:        fullCommit,
		GithubWorkflowRepository: repository,
		GithubWorkflowRef:        ref,
		BuildSignerURI:           workflowURI,
		BuildSignerDigest:        fullCommit,
		RunnerEnvironment:        "github-hosted",
		SourceRepositoryURI:      repositoryURI,
		SourceRepositoryDigest:   fullCommit,
		SourceRepositoryRef:      ref,
		BuildConfigURI:           workflowURI,
		BuildConfigDigest:        fullCommit,
		BuildTrigger:             "push",
	})
	if err != nil {
		return sigstoreReleasePolicy{}, ErrSignedReleaseIdentity
	}
	return sigstoreReleasePolicy{identity: identity}, nil
}

func verifySigstoreChecksumsEvidence(checksums, bundleJSON, trustedRootJSON []byte, trustedRootSHA256 string, policy sigstoreReleasePolicy) error {
	if len(checksums) == 0 || len(checksums) > maxSignedChecksumsBytes || len(bundleJSON) == 0 || len(bundleJSON) > maxSigstoreBundleBytes {
		return ErrSignedReleaseEvidence
	}
	rootDigest := sha256.Sum256(trustedRootJSON)
	if hex.EncodeToString(rootDigest[:]) != trustedRootSHA256 {
		return ErrSignedReleaseEvidence
	}
	trustedRoot, err := root.NewTrustedRootFromJSON(trustedRootJSON)
	if err != nil {
		return ErrSignedReleaseEvidence
	}
	entity := &bundle.Bundle{}
	if err := entity.UnmarshalJSON(bundleJSON); err != nil {
		return ErrSignedReleaseEvidence
	}
	version, err := entity.Version()
	if err != nil || version != "v0.3" || !entity.HasInclusionProof() {
		return ErrSignedReleaseEvidence
	}
	entries, err := entity.TlogEntries()
	if err != nil || len(entries) != 1 {
		return ErrSignedReleaseEvidence
	}
	signature, err := entity.SignatureContent()
	if err != nil || signature.MessageSignatureContent() == nil || signature.EnvelopeContent() != nil {
		return ErrSignedReleaseEvidence
	}
	verifier, err := verify.NewVerifier(
		trustedRoot,
		verify.WithTransparencyLog(1),
		verify.WithObserverTimestamps(1),
		verify.WithSignedCertificateTimestamps(1),
	)
	if err != nil {
		return ErrSignedReleaseEvidence
	}
	_, err = verifier.Verify(entity, verify.NewPolicy(
		verify.WithArtifact(bytes.NewReader(checksums)),
		verify.WithCertificateIdentity(policy.identity),
	))
	if err != nil {
		return ErrSignedReleaseEvidence
	}
	return nil
}

func parseCanonicalMARSSignedChecksums(raw []byte, version string) (map[string][sha256.Size]byte, error) {
	if len(raw) == 0 || len(raw) > maxSignedChecksumsBytes || raw[len(raw)-1] != '\n' || bytes.ContainsAny(raw, "\r\x00") {
		return nil, ErrSignedReleaseChecksums
	}
	expected := expectedMARSArchiveChecksumNames(version)
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) != expectedSignedChecksumCount || len(expected) != expectedSignedChecksumCount {
		return nil, ErrSignedReleaseChecksums
	}
	digests := make(map[string][sha256.Size]byte, expectedSignedChecksumCount)
	for i, line := range lines {
		match := exactChecksumPattern.FindStringSubmatch(line)
		if match == nil || match[2] != expected[i] {
			return nil, ErrSignedReleaseChecksums
		}
		decoded, err := hex.DecodeString(match[1])
		if err != nil || len(decoded) != sha256.Size {
			return nil, ErrSignedReleaseChecksums
		}
		var digest [sha256.Size]byte
		copy(digest[:], decoded)
		if _, duplicate := digests[match[2]]; duplicate {
			return nil, ErrSignedReleaseChecksums
		}
		digests[match[2]] = digest
	}
	return digests, nil
}

func expectedMARSArchiveChecksumNames(version string) []string {
	platforms := [][2]string{{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}}
	names := make([]string, 0, expectedSignedChecksumCount)
	for _, platform := range platforms {
		archive := fmt.Sprintf("mars_%s_%s_%s.tar.gz", version, platform[0], platform[1])
		names = append(names, archive, archive+".sbom.json")
	}
	sort.Strings(names)
	return names
}
