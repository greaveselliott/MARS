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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/githubauth"
)

const (
	signedReleaseAPIBase      = "https://api.github.com/repos/" + DefaultRepoFullName
	maxReleaseMetadataBytes   = 1 << 20
	maxReleaseRefBytes        = 64 << 10
	maxReleaseSBOMBytes       = 4 << 20
	maxReleaseRedirects       = 3
	maxAnnotatedTagDepth      = 4
	releaseMetadataTimeout    = 10 * time.Second
	releaseEvidenceTimeout    = 30 * time.Second
	releaseArchiveTimeout     = 90 * time.Second
	releaseAcquisitionTimeout = 3 * time.Minute
	sigstoreBundleAssetName   = "checksums.txt.sigstore.json"
	releaseChecksumsAssetName = "checksums.txt"
)

var (
	ErrSignedReleaseMetadata         = errors.New("signed release download: immutable release metadata is unavailable or invalid; do not replace the current binary")
	ErrSignedReleaseReplay           = errors.New("signed release download: release ordering or commit identity is unsafe; select an exact trusted version or keep the current binary")
	ErrSignedReleaseDownloadEvidence = errors.New("signed release download: signed checksum evidence is unavailable or invalid; do not replace the current binary")
	ErrSignedReleaseArchive          = errors.New("signed release download: the selected archive is unavailable or invalid; do not replace the current binary")
	ErrSignedReleaseDrift            = errors.New("signed release download: release metadata changed during verification; retry from a stable immutable release")
)

type signedReleaseDownloadRequest struct {
	requestedTag   string
	currentVersion string
	currentCommit  string
	goos           string
	goarch         string
}

type verifiedMARSReleaseDownload struct {
	releaseID   int64
	tag         string
	fullCommit  string
	archiveName string
	authSource  string
	candidate   []byte
}

func (v verifiedMARSReleaseDownload) Binary() []byte {
	return bytes.Clone(v.candidate)
}

type signedDownloadDependencies struct {
	client          *http.Client
	resolveToken    func(context.Context) githubauth.Token
	verifyChecksums func([]byte, []byte, string, string) (SignedChecksums, error)
	verifyArchive   func([]byte, SignedChecksums, string, string, string, string) (VerifiedMARSReleaseArchive, error)
}

func fetchVerifiedMARSRelease(ctx context.Context, client *http.Client, requestedTag, currentVersion, currentCommit, goos, goarch string) (verifiedMARSReleaseDownload, error) {
	deps := signedDownloadDependencies{
		client: client,
		resolveToken: func(ctx context.Context) githubauth.Token {
			return githubauth.ResolveToken(ctx, githubauth.Options{}).Token
		},
		verifyChecksums: VerifyMARSSignedChecksums,
		verifyArchive:   VerifyMARSReleaseArchive,
	}
	return fetchVerifiedMARSReleaseWithDependencies(ctx, signedReleaseDownloadRequest{
		requestedTag: requestedTag, currentVersion: currentVersion, currentCommit: currentCommit,
		goos: goos, goarch: goarch,
	}, deps)
}

func fetchVerifiedMARSReleaseWithDependencies(ctx context.Context, request signedReleaseDownloadRequest, deps signedDownloadDependencies) (verifiedMARSReleaseDownload, error) {
	if ctx == nil || deps.verifyChecksums == nil || deps.verifyArchive == nil {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseMetadata
	}
	ctx, cancel := context.WithTimeout(ctx, releaseAcquisitionTimeout)
	defer cancel()
	requestedTag, latest, ok := normalizeSignedReleaseRequest(request.requestedTag)
	if !ok {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseMetadata
	}
	metadataURL := signedReleaseAPIBase + "/releases/latest"
	if !latest {
		metadataURL = signedReleaseAPIBase + "/releases/tags/" + requestedTag
	}
	session := newSignedDownloadSession(deps)
	release, err := fetchSignedReleaseMetadata(ctx, session, metadataURL)
	if err != nil || (!latest && release.TagName != requestedTag) {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseMetadata
	}
	inventory, assets, err := validateSignedReleaseInventory(release)
	if err != nil {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseMetadata
	}
	resolved, err := resolveSignedReleaseCommit(ctx, session, release.TagName)
	if err != nil {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseMetadata
	}
	if err := enforceSignedReleaseReplayPolicy(request, latest, release.TagName, resolved.commit); err != nil {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseReplay
	}
	archiveName, _, _, ok := marsReleaseArchiveIdentity(release.TagName, resolved.commit, request.goos, request.goarch)
	if !ok {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseMetadata
	}
	archiveAsset, okArchive := assets[archiveName]
	checksumsAsset, okChecksums := assets[releaseChecksumsAssetName]
	bundleAsset, okBundle := assets[sigstoreBundleAssetName]
	if !okArchive || !okChecksums || !okBundle {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseMetadata
	}

	checksums, err := session.get(ctx, checksumsAsset.apiURL(), maxSignedChecksumsBytes, releaseEvidenceTimeout, "application/octet-stream")
	if err != nil || !assetBytesMatch(checksumsAsset, checksums) {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseDownloadEvidence
	}
	bundleJSON, err := session.get(ctx, bundleAsset.apiURL(), maxSigstoreBundleBytes, releaseEvidenceTimeout, "application/octet-stream")
	if err != nil || !assetBytesMatch(bundleAsset, bundleJSON) {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseDownloadEvidence
	}
	authenticated, err := callSignedChecksumsVerifier(deps.verifyChecksums, checksums, bundleJSON, release.TagName, resolved.commit)
	if err != nil || authenticated.Len() != expectedSignedChecksumCount || !authenticated.matchesIdentity(release.TagName, resolved.commit) ||
		!authenticatedReleaseDigestsMatch(authenticated, assets, release.TagName) {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseDownloadEvidence
	}

	archive, err := session.get(ctx, archiveAsset.apiURL(), maxReleaseArchiveBytes, releaseArchiveTimeout, "application/octet-stream")
	if err != nil || !assetBytesMatch(archiveAsset, archive) {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseArchive
	}
	verified, err := callSignedArchiveVerifier(deps.verifyArchive, archive, authenticated, release.TagName, resolved.commit, request.goos, request.goarch)
	if err != nil {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseArchive
	}
	candidate := verified.Binary()
	if len(candidate) == 0 || len(candidate) > maxReleaseBinaryBytes {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseArchive
	}

	finalRelease, err := fetchSignedReleaseMetadata(ctx, session, signedReleaseAPIBase+"/releases/"+strconv.FormatInt(release.ID, 10))
	if err != nil {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseDrift
	}
	finalInventory, _, err := validateSignedReleaseInventory(finalRelease)
	if err != nil || finalRelease.ID != release.ID || finalRelease.TagName != release.TagName || finalInventory != inventory {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseDrift
	}
	if latest {
		finalLatest, err := fetchSignedReleaseMetadata(ctx, session, metadataURL)
		if err != nil || finalLatest.ID != release.ID || finalLatest.TagName != release.TagName {
			return verifiedMARSReleaseDownload{}, ErrSignedReleaseDrift
		}
		latestInventory, _, err := validateSignedReleaseInventory(finalLatest)
		if err != nil || latestInventory != inventory {
			return verifiedMARSReleaseDownload{}, ErrSignedReleaseDrift
		}
	}
	finalResolved, err := resolveSignedReleaseCommit(ctx, session, release.TagName)
	if err != nil || finalResolved != resolved {
		return verifiedMARSReleaseDownload{}, ErrSignedReleaseDrift
	}

	return verifiedMARSReleaseDownload{
		releaseID: release.ID, tag: release.TagName, fullCommit: resolved.commit,
		archiveName: archiveName, authSource: session.authSource, candidate: bytes.Clone(candidate),
	}, nil
}

func normalizeSignedReleaseRequest(value string) (string, bool, bool) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "@"))
	if value == "" || value == DefaultVersion {
		return DefaultVersion, true, true
	}
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	return value, false, exactReleaseTagPattern.MatchString(value)
}

func enforceSignedReleaseReplayPolicy(request signedReleaseDownloadRequest, latest bool, tag, commit string) error {
	currentVersion, stableCurrent := exactStableReleaseVersion(request.currentVersion)
	if latest && !stableCurrent {
		return ErrSignedReleaseReplay
	}
	if !stableCurrent {
		return nil
	}
	relation := CompareVersions(currentVersion, strings.TrimPrefix(tag, "v"))
	if latest && (relation == VersionAhead || relation == VersionUnknown) {
		return ErrSignedReleaseReplay
	}
	if relation == VersionEqual && (!exactCommitPattern.MatchString(request.currentCommit) || request.currentCommit != commit) {
		return ErrSignedReleaseReplay
	}
	return nil
}

func exactStableReleaseVersion(value string) (string, bool) {
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	if strings.Contains(value, "-") || !exactReleaseTagPattern.MatchString("v"+value) {
		return "", false
	}
	return value, true
}

type githubSignedRelease struct {
	ID         int64               `json:"id"`
	TagName    string              `json:"tag_name"`
	Draft      bool                `json:"draft"`
	Prerelease bool                `json:"prerelease"`
	Immutable  bool                `json:"immutable"`
	Assets     []githubSignedAsset `json:"assets"`
}

type githubSignedAsset struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

func (a githubSignedAsset) apiURL() string {
	return signedReleaseAPIBase + "/releases/assets/" + strconv.FormatInt(a.ID, 10)
}

func fetchSignedReleaseMetadata(ctx context.Context, session *signedDownloadSession, endpoint string) (githubSignedRelease, error) {
	raw, err := session.get(ctx, endpoint, maxReleaseMetadataBytes, releaseMetadataTimeout, "application/vnd.github+json")
	if err != nil {
		return githubSignedRelease{}, err
	}
	var release githubSignedRelease
	if err := json.Unmarshal(raw, &release); err != nil {
		return githubSignedRelease{}, ErrSignedReleaseMetadata
	}
	return release, nil
}

func validateSignedReleaseInventory(release githubSignedRelease) ([sha256.Size]byte, map[string]githubSignedAsset, error) {
	if release.ID <= 0 || release.Draft || release.Prerelease || !release.Immutable || !exactReleaseTagPattern.MatchString(release.TagName) {
		return [sha256.Size]byte{}, nil, ErrSignedReleaseMetadata
	}
	version := strings.TrimPrefix(release.TagName, "v")
	expected := append(expectedMARSArchiveChecksumNames(version), releaseChecksumsAssetName, sigstoreBundleAssetName)
	sort.Strings(expected)
	if len(release.Assets) != len(expected) {
		return [sha256.Size]byte{}, nil, ErrSignedReleaseMetadata
	}
	assets := make(map[string]githubSignedAsset, len(expected))
	ids := make(map[int64]struct{}, len(expected))
	for _, asset := range release.Assets {
		limit, ok := signedReleaseAssetLimit(asset.Name, version)
		if !ok || asset.ID <= 0 || asset.State != "uploaded" || asset.Size <= 0 || asset.Size > int64(limit) || !validGitHubAssetDigest(asset.Digest) {
			return [sha256.Size]byte{}, nil, ErrSignedReleaseMetadata
		}
		if _, duplicate := assets[asset.Name]; duplicate {
			return [sha256.Size]byte{}, nil, ErrSignedReleaseMetadata
		}
		if _, duplicate := ids[asset.ID]; duplicate {
			return [sha256.Size]byte{}, nil, ErrSignedReleaseMetadata
		}
		assets[asset.Name] = asset
		ids[asset.ID] = struct{}{}
	}
	for _, name := range expected {
		if _, ok := assets[name]; !ok {
			return [sha256.Size]byte{}, nil, ErrSignedReleaseMetadata
		}
	}
	return signedReleaseInventoryDigest(release, expected, assets), assets, nil
}

func signedReleaseAssetLimit(name, version string) (int, bool) {
	switch name {
	case releaseChecksumsAssetName:
		return maxSignedChecksumsBytes, true
	case sigstoreBundleAssetName:
		return maxSigstoreBundleBytes, true
	}
	for _, archive := range expectedMARSArchiveChecksumNames(version) {
		if name == archive {
			if strings.HasSuffix(name, ".sbom.json") {
				return maxReleaseSBOMBytes, true
			}
			return maxReleaseArchiveBytes, true
		}
	}
	return 0, false
}

func validGitHubAssetDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}

func assetBytesMatch(asset githubSignedAsset, raw []byte) bool {
	if int64(len(raw)) != asset.Size || !validGitHubAssetDigest(asset.Digest) {
		return false
	}
	digest := sha256.Sum256(raw)
	return "sha256:"+hex.EncodeToString(digest[:]) == asset.Digest
}

func authenticatedReleaseDigestsMatch(checksums SignedChecksums, assets map[string]githubSignedAsset, tag string) bool {
	version := strings.TrimPrefix(tag, "v")
	for _, name := range expectedMARSArchiveChecksumNames(version) {
		digest, ok := checksums.Digest(name)
		asset, present := assets[name]
		if !ok || !present || asset.Digest != "sha256:"+hex.EncodeToString(digest[:]) {
			return false
		}
	}
	return true
}

func signedReleaseInventoryDigest(release githubSignedRelease, names []string, assets map[string]githubSignedAsset) [sha256.Size]byte {
	var canonical strings.Builder
	canonical.WriteString(strconv.FormatInt(release.ID, 10))
	canonical.WriteByte('\n')
	canonical.WriteString(release.TagName)
	canonical.WriteByte('\n')
	for _, name := range names {
		asset := assets[name]
		canonical.WriteString(name)
		canonical.WriteByte('\x00')
		canonical.WriteString(strconv.FormatInt(asset.ID, 10))
		canonical.WriteByte('\x00')
		canonical.WriteString(strconv.FormatInt(asset.Size, 10))
		canonical.WriteByte('\x00')
		canonical.WriteString(asset.State)
		canonical.WriteByte('\x00')
		canonical.WriteString(asset.Digest)
		canonical.WriteByte('\n')
	}
	return sha256.Sum256([]byte(canonical.String()))
}

type githubGitObject struct {
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

type githubGitRef struct {
	Ref    string          `json:"ref"`
	Object githubGitObject `json:"object"`
}

type githubTag struct {
	SHA    string          `json:"sha"`
	Object githubGitObject `json:"object"`
}

type resolvedSignedReleaseRef struct {
	commit     string
	chainProof [sha256.Size]byte
}

func resolveSignedReleaseCommit(ctx context.Context, session *signedDownloadSession, tag string) (resolvedSignedReleaseRef, error) {
	if !exactReleaseTagPattern.MatchString(tag) {
		return resolvedSignedReleaseRef{}, ErrSignedReleaseMetadata
	}
	refURL := signedReleaseAPIBase + "/git/ref/tags/" + tag
	raw, err := session.get(ctx, refURL, maxReleaseRefBytes, releaseMetadataTimeout, "application/vnd.github+json")
	if err != nil {
		return resolvedSignedReleaseRef{}, err
	}
	var ref githubGitRef
	if json.Unmarshal(raw, &ref) != nil || ref.Ref != "refs/tags/"+tag || !validGitObject(ref.Object) {
		return resolvedSignedReleaseRef{}, ErrSignedReleaseMetadata
	}
	chain := ref.Ref + "\x00" + ref.Object.Type + "\x00" + ref.Object.SHA
	object := ref.Object
	seen := map[string]struct{}{object.Type + ":" + object.SHA: {}}
	for depth := 0; object.Type == "tag"; depth++ {
		if depth >= maxAnnotatedTagDepth {
			return resolvedSignedReleaseRef{}, ErrSignedReleaseMetadata
		}
		raw, err = session.get(ctx, signedReleaseAPIBase+"/git/tags/"+object.SHA, maxReleaseRefBytes, releaseMetadataTimeout, "application/vnd.github+json")
		if err != nil {
			return resolvedSignedReleaseRef{}, err
		}
		var annotated githubTag
		if json.Unmarshal(raw, &annotated) != nil || annotated.SHA != object.SHA || !validGitObject(annotated.Object) {
			return resolvedSignedReleaseRef{}, ErrSignedReleaseMetadata
		}
		object = annotated.Object
		key := object.Type + ":" + object.SHA
		if _, duplicate := seen[key]; duplicate {
			return resolvedSignedReleaseRef{}, ErrSignedReleaseMetadata
		}
		seen[key] = struct{}{}
		chain += "\x00" + key
	}
	if object.Type != "commit" {
		return resolvedSignedReleaseRef{}, ErrSignedReleaseMetadata
	}
	return resolvedSignedReleaseRef{commit: object.SHA, chainProof: sha256.Sum256([]byte(chain))}, nil
}

func validGitObject(object githubGitObject) bool {
	return (object.Type == "commit" || object.Type == "tag") && exactCommitPattern.MatchString(object.SHA)
}

type signedDownloadSession struct {
	client        *http.Client
	resolveToken  func(context.Context) githubauth.Token
	tokenResolved bool
	token         githubauth.Token
	authSource    string
}

func newSignedDownloadSession(deps signedDownloadDependencies) *signedDownloadSession {
	client := deps.client
	if client == nil {
		client = &http.Client{}
	}
	cloned := *client
	cloned.Jar = nil
	cloned.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &signedDownloadSession{client: &cloned, resolveToken: deps.resolveToken, authSource: githubauth.SourceNone}
}

func (s *signedDownloadSession) get(ctx context.Context, endpoint string, limit int, timeout time.Duration, accept string) ([]byte, error) {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	allowRedirect := exactGitHubAssetAPIURL(endpoint)
	raw, status, err := s.getOnce(requestCtx, endpoint, limit, accept, "", allowRedirect)
	if err == nil && status >= 200 && status < 300 {
		return raw, nil
	}
	if !isGitHubAuthFallbackStatus(status) || !exactGitHubAPIURL(endpoint) {
		return nil, ErrSignedReleaseMetadata
	}
	token := s.optionalToken(requestCtx)
	if token.Value == "" {
		return nil, ErrSignedReleaseMetadata
	}
	raw, status, err = s.getOnce(requestCtx, endpoint, limit, accept, token.Value, allowRedirect)
	if err != nil || status < 200 || status >= 300 {
		return nil, ErrSignedReleaseMetadata
	}
	s.authSource = token.Source
	return raw, nil
}

func (s *signedDownloadSession) optionalToken(ctx context.Context) githubauth.Token {
	if s.tokenResolved {
		return s.token
	}
	s.tokenResolved = true
	if s.resolveToken != nil {
		s.token = callSignedReleaseTokenResolver(s.resolveToken, ctx)
	}
	return s.token
}

func callSignedReleaseTokenResolver(fn func(context.Context) githubauth.Token, ctx context.Context) (token githubauth.Token) {
	defer func() {
		if recover() != nil {
			token = githubauth.Token{Source: githubauth.SourceNone}
		}
	}()
	token = fn(ctx)
	if strings.TrimSpace(token.Value) != token.Value || strings.ContainsAny(token.Value, "\r\n\x00") {
		return githubauth.Token{Source: githubauth.SourceNone}
	}
	return token
}

func (s *signedDownloadSession) getOnce(ctx context.Context, endpoint string, limit int, accept, token string, allowRedirect bool) ([]byte, int, error) {
	current, err := url.Parse(endpoint)
	if err != nil || current.User != nil || current.Scheme != "https" {
		return nil, 0, ErrSignedReleaseMetadata
	}
	for redirects := 0; ; redirects++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, current.String(), nil)
		if err != nil {
			return nil, 0, ErrSignedReleaseMetadata
		}
		req.Header.Set("Accept", accept)
		req.Header.Set("User-Agent", "mars-signed-release-download")
		req.Header.Del("Cookie")
		req.Header.Del("Proxy-Authorization")
		if redirects == 0 && token != "" && exactGitHubAPIURL(current.String()) {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if exactGitHubAPIURL(current.String()) {
			req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
		}
		resp, err := s.client.Do(req)
		if err != nil {
			return nil, 0, ErrSignedReleaseMetadata
		}
		if isHTTPRedirect(resp.StatusCode) {
			_ = resp.Body.Close()
			if !allowRedirect || redirects >= maxReleaseRedirects {
				return nil, resp.StatusCode, ErrSignedReleaseMetadata
			}
			next, err := resp.Location()
			if err != nil || next.Scheme != "https" || next.User != nil {
				return nil, resp.StatusCode, ErrSignedReleaseMetadata
			}
			current = next
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			_ = resp.Body.Close()
			return nil, resp.StatusCode, nil
		}
		if resp.ContentLength > int64(limit) {
			_ = resp.Body.Close()
			return nil, resp.StatusCode, ErrSignedReleaseMetadata
		}
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
		closeErr := resp.Body.Close()
		if readErr != nil || closeErr != nil || len(raw) == 0 || len(raw) > limit {
			return nil, resp.StatusCode, ErrSignedReleaseMetadata
		}
		return raw, resp.StatusCode, nil
	}
}

func exactGitHubAPIURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host == "api.github.com" && parsed.User == nil
}

func exactGitHubAssetAPIURL(value string) bool {
	if !strings.HasPrefix(value, signedReleaseAPIBase+"/releases/assets/") || !exactGitHubAPIURL(value) {
		return false
	}
	id := strings.TrimPrefix(value, signedReleaseAPIBase+"/releases/assets/")
	_, err := strconv.ParseInt(id, 10, 64)
	return id != "" && err == nil
}

func isGitHubAuthFallbackStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusNotFound
}

func isHTTPRedirect(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func callSignedChecksumsVerifier(fn func([]byte, []byte, string, string) (SignedChecksums, error), checksums, bundle []byte, tag, commit string) (result SignedChecksums, err error) {
	defer func() {
		if recover() != nil {
			result, err = SignedChecksums{}, ErrSignedReleaseDownloadEvidence
		}
	}()
	return fn(bytes.Clone(checksums), bytes.Clone(bundle), tag, commit)
}

func callSignedArchiveVerifier(fn func([]byte, SignedChecksums, string, string, string, string) (VerifiedMARSReleaseArchive, error), archive []byte, checksums SignedChecksums, tag, commit, goos, goarch string) (result VerifiedMARSReleaseArchive, err error) {
	defer func() {
		if recover() != nil {
			result, err = VerifiedMARSReleaseArchive{}, ErrSignedReleaseArchive
		}
	}()
	return fn(bytes.Clone(archive), checksums, tag, commit, goos, goarch)
}
