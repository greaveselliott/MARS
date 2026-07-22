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
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/stretchr/testify/require"
)

const (
	goreleaserFixtureCommit = "770a4fc7a8fb2dca874b6c98cb739dd64fc931c0"
	goreleaserFixtureTag    = "v2.17.0"
)

//go:embed testdata/goreleaser-v2.17.0-checksums.txt
var goreleaserFixtureChecksums []byte

//go:embed testdata/goreleaser-v2.17.0-checksums.txt.sigstore.json
var goreleaserFixtureBundle []byte

func TestVerifySigstoreChecksumsEvidenceRealOfflineFixture(t *testing.T) {
	policy, err := newSigstoreReleasePolicy("goreleaser/goreleaser", "release.yml", goreleaserFixtureTag, goreleaserFixtureCommit)
	require.NoError(t, err)
	require.NoError(t, verifySigstoreChecksumsEvidence(
		goreleaserFixtureChecksums,
		goreleaserFixtureBundle,
		pinnedTrustedRootJSON,
		pinnedTrustedRootSHA256,
		policy,
	))

	t.Run("tampered exact artifact bytes", func(t *testing.T) {
		tampered := append([]byte(nil), goreleaserFixtureChecksums...)
		tampered[0] ^= 1
		require.ErrorIs(t, verifySigstoreChecksumsEvidence(tampered, goreleaserFixtureBundle, pinnedTrustedRootJSON, pinnedTrustedRootSHA256, policy), ErrSignedReleaseEvidence)
	})

	t.Run("wrong source commit", func(t *testing.T) {
		wrong, err := newSigstoreReleasePolicy("goreleaser/goreleaser", "release.yml", goreleaserFixtureTag, strings.Repeat("a", 40))
		require.NoError(t, err)
		require.ErrorIs(t, verifySigstoreChecksumsEvidence(goreleaserFixtureChecksums, goreleaserFixtureBundle, pinnedTrustedRootJSON, pinnedTrustedRootSHA256, wrong), ErrSignedReleaseEvidence)
	})

	t.Run("wrong trusted root pin", func(t *testing.T) {
		require.ErrorIs(t, verifySigstoreChecksumsEvidence(goreleaserFixtureChecksums, goreleaserFixtureBundle, pinnedTrustedRootJSON, strings.Repeat("0", 64), policy), ErrSignedReleaseEvidence)
	})

	t.Run("missing inclusion proof", func(t *testing.T) {
		var document map[string]any
		require.NoError(t, json.Unmarshal(goreleaserFixtureBundle, &document))
		material := document["verificationMaterial"].(map[string]any)
		entries := material["tlogEntries"].([]any)
		entries[0].(map[string]any)["inclusionProof"] = nil
		withoutProof, err := json.Marshal(document)
		require.NoError(t, err)
		require.ErrorIs(t, verifySigstoreChecksumsEvidence(goreleaserFixtureChecksums, withoutProof, pinnedTrustedRootJSON, pinnedTrustedRootSHA256, policy), ErrSignedReleaseEvidence)
	})

	t.Run("unsupported bundle version", func(t *testing.T) {
		unsupported := strings.Replace(string(goreleaserFixtureBundle), "application/vnd.dev.sigstore.bundle.v0.3+json", "application/vnd.dev.sigstore.bundle.v0.2+json", 1)
		require.ErrorIs(t, verifySigstoreChecksumsEvidence(goreleaserFixtureChecksums, []byte(unsupported), pinnedTrustedRootJSON, pinnedTrustedRootSHA256, policy), ErrSignedReleaseEvidence)
	})
}

func TestParseCanonicalMARSSignedChecksums(t *testing.T) {
	const version = "0.69.0"
	raw := canonicalMARSChecksumsFixture(version)
	digests, err := parseCanonicalMARSSignedChecksums(raw, version)
	require.NoError(t, err)
	require.Len(t, digests, expectedSignedChecksumCount)

	name := expectedMARSArchiveChecksumNames(version)[0]
	want := sha256.Sum256([]byte(name))
	require.Equal(t, want, digests[name])
	raw[0] ^= 1
	require.Equal(t, want, digests[name], "result must not alias caller-owned bytes")
}

func TestParseCanonicalMARSSignedChecksumsRejectsNoncanonicalInput(t *testing.T) {
	const version = "0.69.0"
	valid := canonicalMARSChecksumsFixture(version)
	lines := strings.Split(strings.TrimSuffix(string(valid), "\n"), "\n")
	tests := map[string][]byte{
		"missing final LF": []byte(strings.TrimSuffix(string(valid), "\n")),
		"CRLF":             []byte(strings.ReplaceAll(string(valid), "\n", "\r\n")),
		"missing record":   []byte(strings.Join(lines[:len(lines)-1], "\n") + "\n"),
		"extra record":     append(append([]byte(nil), valid...), []byte(lines[0]+"\n")...),
		"duplicate record": []byte(strings.Join(append(append([]string(nil), lines[:7]...), lines[0]), "\n") + "\n"),
		"unsorted records": []byte(strings.Join(append([]string{lines[1], lines[0]}, lines[2:]...), "\n") + "\n"),
		"uppercase digest": []byte(strings.ToUpper(lines[0][:64]) + lines[0][64:] + "\n" + strings.Join(lines[1:], "\n") + "\n"),
		"one space":        []byte(strings.Replace(string(valid), "  ", " ", 1)),
		"star separator":   []byte(strings.Replace(string(valid), "  ", " *", 1)),
		"wrong version":    []byte(strings.ReplaceAll(string(valid), version, "0.69.1")),
		"path separator":   []byte(strings.Replace(string(valid), "mars_", "nested/mars_", 1)),
		"oversized":        []byte(strings.Repeat("x", maxSignedChecksumsBytes+1)),
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parseCanonicalMARSSignedChecksums(raw, version)
			require.ErrorIs(t, err, ErrSignedReleaseChecksums)
		})
	}
}

func TestMARSSigstorePolicyBindsExactWorkflowAndCommit(t *testing.T) {
	const tag = "v0.69.0"
	commit := strings.Repeat("a", 40)
	policy, err := newSigstoreReleasePolicy(marsReleaseRepository, marsReleaseWorkflow, tag, commit)
	require.NoError(t, err)
	workflowURI := "https://github.com/greaveselliott/MARS/.github/workflows/release.yml@refs/tags/" + tag
	summary := certificate.Summary{
		SubjectAlternativeName: workflowURI,
		Extensions: certificate.Extensions{
			Issuer:                   sigstoreActionsIssuer,
			GithubWorkflowSHA:        commit,
			GithubWorkflowRepository: marsReleaseRepository,
			GithubWorkflowRef:        "refs/tags/" + tag,
			BuildSignerURI:           workflowURI,
			BuildSignerDigest:        commit,
			RunnerEnvironment:        "github-hosted",
			SourceRepositoryURI:      "https://github.com/greaveselliott/MARS",
			SourceRepositoryDigest:   commit,
			SourceRepositoryRef:      "refs/tags/" + tag,
			BuildConfigURI:           workflowURI,
			BuildConfigDigest:        commit,
			BuildTrigger:             "push",
		},
	}
	require.NoError(t, policy.identity.Verify(summary))
	summary.GithubWorkflowSHA = strings.Repeat("b", 40)
	require.Error(t, policy.identity.Verify(summary))
	summary.GithubWorkflowSHA = commit
	summary.SourceRepositoryDigest = strings.Repeat("b", 40)
	require.Error(t, policy.identity.Verify(summary))
}

func TestVerifyMARSSignedChecksumsErrorsAreFixedAndRedacted(t *testing.T) {
	const hostile = "https://attacker.invalid/TOKEN-DO-NOT-LOG"
	checksums := canonicalMARSChecksumsFixture("0.69.0")
	_, err := VerifyMARSSignedChecksums(checksums, []byte(`{"hostile":"`+hostile+`"}`), "v0.69.0", strings.Repeat("a", 40))
	require.ErrorIs(t, err, ErrSignedReleaseEvidence)
	require.NotContains(t, err.Error(), hostile)
	require.NotContains(t, err.Error(), "TOKEN-DO-NOT-LOG")

	_, err = VerifyMARSSignedChecksums(checksums, goreleaserFixtureBundle, "0.69.0", strings.Repeat("a", 40))
	require.ErrorIs(t, err, ErrSignedReleaseIdentity)
}

func canonicalMARSChecksumsFixture(version string) []byte {
	var builder strings.Builder
	for _, name := range expectedMARSArchiveChecksumNames(version) {
		digest := sha256.Sum256([]byte(name))
		fmt.Fprintf(&builder, "%s  %s\n", hex.EncodeToString(digest[:]), name)
	}
	return []byte(builder.String())
}
