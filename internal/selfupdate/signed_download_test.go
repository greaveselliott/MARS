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
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/greaveselliott/mars/internal/githubauth"
	"github.com/stretchr/testify/require"
)

const (
	testDownloadTag           = "v0.69.0"
	testDownloadCommit        = "0123456789abcdef0123456789abcdef01234567"
	testDownloadOtherCommit   = "abcdef0123456789abcdef0123456789abcdef01"
	testDownloadReleaseID     = int64(690)
	testDownloadCurrentStable = "0.68.49"
)

func TestFetchVerifiedMARSReleaseHappyPublicLightweightTag(t *testing.T) {
	harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
	harness.checkRequest = func(req *http.Request) {
		require.Empty(t, req.Header.Get("Authorization"))
		require.Empty(t, req.Header.Get("Cookie"))
		require.Empty(t, req.Header.Get("Proxy-Authorization"))
	}

	got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
		requestedTag: DefaultVersion, currentVersion: testDownloadCurrentStable,
		goos: "darwin", goarch: "arm64",
	}, harness.dependencies())
	require.NoError(t, err)
	require.Equal(t, testDownloadReleaseID, got.releaseID)
	require.Equal(t, testDownloadTag, got.tag)
	require.Equal(t, testDownloadCommit, got.fullCommit)
	require.Equal(t, harness.fixture.archiveName, got.archiveName)
	require.Equal(t, githubauth.SourceNone, got.authSource)
	require.Zero(t, harness.tokenCalls, "public success must not resolve private credentials")
	require.Equal(t, 1, harness.checksumVerifierCalls)
	require.Equal(t, 1, harness.archiveVerifierCalls)

	wantEvents := []string{
		signedLatestPath(), signedRefPath(testDownloadTag),
		harness.fixture.assetPath(releaseChecksumsAssetName), harness.fixture.assetPath(sigstoreBundleAssetName),
		"verify-checksums", harness.fixture.assetPath(harness.fixture.archiveName), "verify-archive",
		signedReleaseIDPath(testDownloadReleaseID), signedLatestPath(), signedRefPath(testDownloadTag),
	}
	require.Equal(t, wantEvents, harness.events)
	var assetRequests []string
	for _, event := range harness.events {
		if strings.Contains(event, "/releases/assets/") {
			assetRequests = append(assetRequests, event)
		}
	}
	require.Equal(t, []string{
		harness.fixture.assetPath(releaseChecksumsAssetName),
		harness.fixture.assetPath(sigstoreBundleAssetName),
		harness.fixture.assetPath(harness.fixture.archiveName),
	}, assetRequests, "only checksums, signature bundle, and the selected archive may download")

	wantCandidate := append([]byte(nil), harness.candidate...)
	harness.candidate[0] ^= 1
	first := got.Binary()
	require.Equal(t, wantCandidate, first, "download result must not alias verifier-owned bytes")
	first[0] ^= 1
	require.Equal(t, wantCandidate, got.Binary(), "Binary must return an independent clone")
}

func TestFetchVerifiedMARSReleaseAcceptsBoundedAnnotatedTagAndExplicitOlderVersion(t *testing.T) {
	harness := newSignedDownloadHarness(t, "v0.68.0", "linux", "amd64")
	tagObject := strings.Repeat("1", 40)
	harness.refObjects = []githubGitObject{{Type: "tag", SHA: tagObject}}
	harness.annotatedObjects[tagObject] = githubGitObject{Type: "commit", SHA: testDownloadCommit}

	got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
		requestedTag: "v0.68.0", currentVersion: "0.69.0", currentCommit: testDownloadOtherCommit,
		goos: "linux", goarch: "amd64",
	}, harness.dependencies())
	require.NoError(t, err, "an explicit older version is an authenticated rollback request")
	require.Equal(t, testDownloadCommit, got.fullCommit)
	require.Equal(t, 2, harness.annotatedCalls[tagObject], "initial and final ref observations must both follow the annotated tag")
	require.Equal(t, testDownloadCommit, harness.archiveCommit)
}

func TestFetchVerifiedMARSReleaseRejectsMetadataContractMutations(t *testing.T) {
	tests := map[string]func(*githubSignedRelease){
		"tag":        func(release *githubSignedRelease) { release.TagName = "v01.2.3" },
		"draft":      func(release *githubSignedRelease) { release.Draft = true },
		"prerelease": func(release *githubSignedRelease) { release.Prerelease = true },
		"mutable":    func(release *githubSignedRelease) { release.Immutable = false },
		"inventory":  func(release *githubSignedRelease) { release.Assets = release.Assets[:len(release.Assets)-1] },
		"release id": func(release *githubSignedRelease) { release.ID = 0 },
		"asset id":   func(release *githubSignedRelease) { release.Assets[0].ID = 0 },
		"state":      func(release *githubSignedRelease) { release.Assets[0].State = "new" },
		"size":       func(release *githubSignedRelease) { release.Assets[0].Size = 0 },
		"digest":     func(release *githubSignedRelease) { release.Assets[0].Digest = "sha256:" + strings.Repeat("A", 64) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
			mutate(&harness.release)
			got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
				requestedTag: DefaultVersion, currentVersion: testDownloadCurrentStable,
				goos: "darwin", goarch: "arm64",
			}, harness.dependencies())
			require.ErrorIs(t, err, ErrSignedReleaseMetadata)
			require.Empty(t, got.Binary())
			require.Equal(t, []string{signedLatestPath()}, harness.events, "metadata rejection must precede refs, assets, and verifiers")
			require.Zero(t, harness.checksumVerifierCalls)
			require.Zero(t, harness.archiveVerifierCalls)
			require.Zero(t, harness.tokenCalls)
		})
	}
}

func TestFetchVerifiedMARSReleaseVerificationFailuresStopProgress(t *testing.T) {
	t.Run("A1 failure does not fetch archive", func(t *testing.T) {
		harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
		harness.checksumVerifierErr = errors.New("synthetic A1 rejection")
		got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
			requestedTag: DefaultVersion, currentVersion: testDownloadCurrentStable,
			goos: "darwin", goarch: "arm64",
		}, harness.dependencies())
		require.ErrorIs(t, err, ErrSignedReleaseDownloadEvidence)
		require.Empty(t, got.Binary())
		require.Equal(t, []string{
			signedLatestPath(), signedRefPath(testDownloadTag),
			harness.fixture.assetPath(releaseChecksumsAssetName), harness.fixture.assetPath(sigstoreBundleAssetName),
			"verify-checksums",
		}, harness.events)
		require.Zero(t, harness.archiveVerifierCalls)
	})

	t.Run("A2 failure returns no candidate", func(t *testing.T) {
		harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
		harness.archiveVerifierErr = errors.New("synthetic A2 rejection")
		got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
			requestedTag: DefaultVersion, currentVersion: testDownloadCurrentStable,
			goos: "darwin", goarch: "arm64",
		}, harness.dependencies())
		require.ErrorIs(t, err, ErrSignedReleaseArchive)
		require.Empty(t, got.Binary())
		require.Equal(t, 1, harness.checksumVerifierCalls)
		require.Equal(t, 1, harness.archiveVerifierCalls)
		require.NotContains(t, harness.events, signedReleaseIDPath(testDownloadReleaseID), "drift checks must not run after A2 rejects")
	})
}

func TestFetchVerifiedMARSReleaseRejectsAuthenticatedDigestMetadataMismatch(t *testing.T) {
	tests := map[string]func(*signedDownloadHarness) string{
		"unselected archive": func(harness *signedDownloadHarness) string {
			for _, name := range expectedMARSArchiveChecksumNames(strings.TrimPrefix(testDownloadTag, "v")) {
				if !strings.HasSuffix(name, ".sbom.json") && name != harness.fixture.archiveName {
					return name
				}
			}
			t.Fatal("fixture has no unselected archive")
			return ""
		},
		"SBOM": func(harness *signedDownloadHarness) string {
			for _, name := range expectedMARSArchiveChecksumNames(strings.TrimPrefix(testDownloadTag, "v")) {
				if strings.HasSuffix(name, ".sbom.json") {
					return name
				}
			}
			t.Fatal("fixture has no SBOM")
			return ""
		},
	}
	for name, selectAsset := range tests {
		t.Run(name, func(t *testing.T) {
			harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
			assetName := selectAsset(harness)
			for index := range harness.release.Assets {
				if harness.release.Assets[index].Name == assetName {
					harness.release.Assets[index].Digest = "sha256:" + strings.Repeat("e", 64)
					break
				}
			}

			got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
				requestedTag: DefaultVersion, currentVersion: testDownloadCurrentStable,
				goos: "darwin", goarch: "arm64",
			}, harness.dependencies())
			require.ErrorIs(t, err, ErrSignedReleaseDownloadEvidence)
			require.Equal(t, ErrSignedReleaseDownloadEvidence.Error(), err.Error(), "failure must remain fixed and redacted")
			require.NotContains(t, err.Error(), assetName)
			require.Empty(t, got.Binary())
			require.Equal(t, 1, harness.checksumVerifierCalls)
			require.Zero(t, harness.archiveVerifierCalls)
			require.Equal(t, "verify-checksums", harness.events[len(harness.events)-1])
			require.NotContains(t, harness.events, harness.fixture.assetPath(harness.fixture.archiveName), "digest mismatch must reject before the selected archive download")
		})
	}
}

func TestFetchVerifiedMARSReleaseRejectsFinalReleaseAndRefDrift(t *testing.T) {
	t.Run("release inventory", func(t *testing.T) {
		harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
		final := cloneGitHubSignedRelease(harness.release)
		for index := range final.Assets {
			if strings.HasSuffix(final.Assets[index].Name, ".sbom.json") {
				final.Assets[index].Digest = "sha256:" + strings.Repeat("f", 64)
				break
			}
		}
		harness.finalRelease = &final
		got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
			requestedTag: testDownloadTag, goos: "darwin", goarch: "arm64",
		}, harness.dependencies())
		require.ErrorIs(t, err, ErrSignedReleaseDrift)
		require.Empty(t, got.Binary())
		require.Equal(t, 1, harness.archiveVerifierCalls)
		require.Equal(t, signedReleaseIDPath(testDownloadReleaseID), harness.events[len(harness.events)-1])
	})

	t.Run("tag ref", func(t *testing.T) {
		harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
		harness.refObjects = []githubGitObject{
			{Type: "commit", SHA: testDownloadCommit},
			{Type: "commit", SHA: testDownloadOtherCommit},
		}
		got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
			requestedTag: testDownloadTag, goos: "darwin", goarch: "arm64",
		}, harness.dependencies())
		require.ErrorIs(t, err, ErrSignedReleaseDrift)
		require.Empty(t, got.Binary())
		require.Equal(t, 2, harness.refCalls)
		require.Equal(t, signedRefPath(testDownloadTag), harness.events[len(harness.events)-1])
	})
}

func TestSignedReleaseReplayPolicy(t *testing.T) {
	tests := []struct {
		name    string
		request signedReleaseDownloadRequest
		latest  bool
		wantErr bool
	}{
		{name: "latest downgrade", request: signedReleaseDownloadRequest{currentVersion: "0.70.0", currentCommit: testDownloadOtherCommit}, latest: true, wantErr: true},
		{name: "unknown current version", request: signedReleaseDownloadRequest{currentVersion: "0.69.0-dev", currentCommit: testDownloadCommit}, latest: true, wantErr: true},
		{name: "same version different commit", request: signedReleaseDownloadRequest{currentVersion: "0.69.0", currentCommit: testDownloadOtherCommit}, latest: true, wantErr: true},
		{name: "explicit older allowed", request: signedReleaseDownloadRequest{currentVersion: "0.70.0", currentCommit: testDownloadOtherCommit}, latest: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := enforceSignedReleaseReplayPolicy(test.request, test.latest, testDownloadTag, testDownloadCommit)
			if test.wantErr {
				require.ErrorIs(t, err, ErrSignedReleaseReplay)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestFetchVerifiedMARSReleaseEnforcesDownloadBoundaries(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		oversizeLimit int
		wantErr       error
		wantA1        int
	}{
		{name: "checksums quota", assetName: releaseChecksumsAssetName, oversizeLimit: maxSignedChecksumsBytes, wantErr: ErrSignedReleaseDownloadEvidence},
		{name: "bundle quota", assetName: sigstoreBundleAssetName, oversizeLimit: maxSigstoreBundleBytes, wantErr: ErrSignedReleaseDownloadEvidence},
		{name: "archive quota", assetName: "archive", oversizeLimit: maxReleaseArchiveBytes, wantErr: ErrSignedReleaseArchive, wantA1: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
			assetName := test.assetName
			if assetName == "archive" {
				assetName = harness.fixture.archiveName
			}
			id := harness.fixture.asset(assetName).ID
			harness.assetResponse[id] = func(req *http.Request, body []byte) *http.Response {
				response := signedDownloadResponse(req, http.StatusOK, body)
				response.ContentLength = int64(test.oversizeLimit) + 1
				return response
			}
			got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
				requestedTag: DefaultVersion, currentVersion: testDownloadCurrentStable,
				goos: "darwin", goarch: "arm64",
			}, harness.dependencies())
			require.ErrorIs(t, err, test.wantErr)
			require.Empty(t, got.Binary())
			require.Equal(t, test.wantA1, harness.checksumVerifierCalls)
			require.Zero(t, harness.archiveVerifierCalls)
			require.Equal(t, harness.fixture.assetPath(assetName), harness.events[len(harness.events)-1])
		})
	}

	t.Run("short asset", func(t *testing.T) {
		harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
		for index := range harness.release.Assets {
			if harness.release.Assets[index].Name == releaseChecksumsAssetName {
				harness.release.Assets[index].Size++
				break
			}
		}
		got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
			requestedTag: DefaultVersion, currentVersion: testDownloadCurrentStable,
			goos: "darwin", goarch: "arm64",
		}, harness.dependencies())
		require.ErrorIs(t, err, ErrSignedReleaseDownloadEvidence)
		require.Empty(t, got.Binary())
		require.Zero(t, harness.checksumVerifierCalls, "size mismatch must reject before A1")
	})

	t.Run("non-success status", func(t *testing.T) {
		harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
		harness.metadataResponse = func(req *http.Request) *http.Response {
			return signedDownloadResponse(req, http.StatusInternalServerError, []byte("TOKEN-DO-NOT-LOG"))
		}
		got, err := fetchVerifiedMARSReleaseWithDependencies(context.Background(), signedReleaseDownloadRequest{
			requestedTag: DefaultVersion, currentVersion: testDownloadCurrentStable,
			goos: "darwin", goarch: "arm64",
		}, harness.dependencies())
		require.ErrorIs(t, err, ErrSignedReleaseMetadata)
		require.NotContains(t, err.Error(), "TOKEN-DO-NOT-LOG")
		require.Empty(t, got.Binary())
		require.Zero(t, harness.tokenCalls, "500 must not trigger private-auth fallback")
		require.Equal(t, []string{signedLatestPath()}, harness.events)
	})

	t.Run("cancelled", func(t *testing.T) {
		harness := newSignedDownloadHarness(t, testDownloadTag, "darwin", "arm64")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		got, err := fetchVerifiedMARSReleaseWithDependencies(ctx, signedReleaseDownloadRequest{
			requestedTag: DefaultVersion, currentVersion: testDownloadCurrentStable,
			goos: "darwin", goarch: "arm64",
		}, harness.dependencies())
		require.ErrorIs(t, err, ErrSignedReleaseMetadata)
		require.Empty(t, got.Binary())
		require.Zero(t, harness.checksumVerifierCalls)
		require.Zero(t, harness.archiveVerifierCalls)
	})
}

func TestSignedDownloadPrivateFallbackAndManualRedirectStripSensitiveHeaders(t *testing.T) {
	const (
		assetURL  = signedReleaseAPIBase + "/releases/assets/42"
		objectURL = "https://objects.githubusercontent.com/release/asset"
		token     = "ghp_TOKEN_DO_NOT_LOG"
	)
	var events []string
	apiCalls := 0
	client := fakeHTTPClient(func(req *http.Request) (*http.Response, error) {
		events = append(events, req.URL.String())
		require.Empty(t, req.Header.Get("Cookie"))
		require.Empty(t, req.Header.Get("Proxy-Authorization"))
		switch req.URL.String() {
		case assetURL:
			apiCalls++
			if apiCalls == 1 {
				require.Empty(t, req.Header.Get("Authorization"), "anonymous access must be attempted first")
				return signedDownloadResponse(req, http.StatusNotFound, []byte("private")), nil
			}
			require.Equal(t, "Bearer "+token, req.Header.Get("Authorization"))
			response := signedDownloadResponse(req, http.StatusFound, []byte("redirect"))
			response.Header.Set("Location", objectURL)
			return response, nil
		case objectURL:
			require.Empty(t, req.Header.Get("Authorization"), "manual cross-origin redirect must strip the token")
			require.Empty(t, req.Header.Get("X-GitHub-Api-Version"))
			return signedDownloadResponse(req, http.StatusOK, []byte("asset")), nil
		default:
			return nil, errors.New("unexpected request")
		}
	})
	resolveCalls := 0
	session := newSignedDownloadSession(signedDownloadDependencies{
		client: client,
		resolveToken: func(context.Context) githubauth.Token {
			resolveCalls++
			return githubauth.Token{Value: token, Source: githubauth.SourceEnvGHToken}
		},
	})
	raw, err := session.get(context.Background(), assetURL, 64, time.Second, "application/octet-stream")
	require.NoError(t, err)
	require.Equal(t, []byte("asset"), raw)
	require.Equal(t, 1, resolveCalls)
	require.Equal(t, githubauth.SourceEnvGHToken, session.authSource)
	require.Equal(t, []string{assetURL, assetURL, objectURL}, events)
}

type signedDownloadFixture struct {
	release     githubSignedRelease
	bodies      map[int64][]byte
	byName      map[string]githubSignedAsset
	archiveName string
}

func newSignedDownloadFixture(t *testing.T, tag, goos, goarch string) signedDownloadFixture {
	t.Helper()
	version := strings.TrimPrefix(tag, "v")
	archiveName, _, _, ok := marsReleaseArchiveIdentity(tag, testDownloadCommit, goos, goarch)
	require.True(t, ok)
	names := append(expectedMARSArchiveChecksumNames(version), releaseChecksumsAssetName, sigstoreBundleAssetName)
	sort.Strings(names)
	fixture := signedDownloadFixture{
		release: githubSignedRelease{ID: testDownloadReleaseID, TagName: tag, Immutable: true},
		bodies:  make(map[int64][]byte, len(names)), byName: make(map[string]githubSignedAsset, len(names)),
		archiveName: archiveName,
	}
	for index, name := range names {
		id := int64(1000 + index)
		body := []byte("fixture:" + name)
		digest := sha256.Sum256(body)
		asset := githubSignedAsset{
			ID: id, Name: name, State: "uploaded", Size: int64(len(body)),
			Digest: "sha256:" + hex.EncodeToString(digest[:]),
		}
		fixture.release.Assets = append(fixture.release.Assets, asset)
		fixture.bodies[id] = append([]byte(nil), body...)
		fixture.byName[name] = asset
	}
	return fixture
}

func (f signedDownloadFixture) asset(name string) githubSignedAsset {
	return f.byName[name]
}

func (f signedDownloadFixture) assetPath(name string) string {
	return "/repos/greaveselliott/MARS/releases/assets/" + strconv.FormatInt(f.asset(name).ID, 10)
}

type signedDownloadHarness struct {
	t                     *testing.T
	fixture               signedDownloadFixture
	release               githubSignedRelease
	finalRelease          *githubSignedRelease
	refObjects            []githubGitObject
	annotatedObjects      map[string]githubGitObject
	annotatedCalls        map[string]int
	assetResponse         map[int64]func(*http.Request, []byte) *http.Response
	metadataResponse      func(*http.Request) *http.Response
	checkRequest          func(*http.Request)
	events                []string
	latestCalls           int
	refCalls              int
	tokenCalls            int
	token                 githubauth.Token
	checksumVerifierErr   error
	archiveVerifierErr    error
	checksumVerifierCalls int
	archiveVerifierCalls  int
	checksumTag           string
	checksumCommit        string
	archiveTag            string
	archiveCommit         string
	candidate             []byte
}

func newSignedDownloadHarness(t *testing.T, tag, goos, goarch string) *signedDownloadHarness {
	t.Helper()
	fixture := newSignedDownloadFixture(t, tag, goos, goarch)
	return &signedDownloadHarness{
		t: t, fixture: fixture, release: cloneGitHubSignedRelease(fixture.release),
		refObjects:       []githubGitObject{{Type: "commit", SHA: testDownloadCommit}},
		annotatedObjects: make(map[string]githubGitObject), annotatedCalls: make(map[string]int),
		assetResponse: make(map[int64]func(*http.Request, []byte) *http.Response),
		candidate:     []byte("verified candidate"),
	}
}

func (h *signedDownloadHarness) dependencies() signedDownloadDependencies {
	return signedDownloadDependencies{
		client: fakeHTTPClient(h.roundTrip),
		resolveToken: func(context.Context) githubauth.Token {
			h.tokenCalls++
			return h.token
		},
		verifyChecksums: func(checksums, bundle []byte, tag, commit string) (SignedChecksums, error) {
			h.events = append(h.events, "verify-checksums")
			h.checksumVerifierCalls++
			h.checksumTag, h.checksumCommit = tag, commit
			require.Equal(h.t, h.fixture.bodies[h.fixture.asset(releaseChecksumsAssetName).ID], checksums)
			require.Equal(h.t, h.fixture.bodies[h.fixture.asset(sigstoreBundleAssetName).ID], bundle)
			if h.checksumVerifierErr != nil {
				return SignedChecksums{}, h.checksumVerifierErr
			}
			digests := make(map[string][sha256.Size]byte, expectedSignedChecksumCount)
			for _, name := range expectedMARSArchiveChecksumNames(strings.TrimPrefix(tag, "v")) {
				digests[name] = sha256.Sum256(h.fixture.bodies[h.fixture.asset(name).ID])
			}
			return SignedChecksums{digests: digests, tag: tag, fullCommit: commit}, nil
		},
		verifyArchive: func(archive []byte, checksums SignedChecksums, tag, commit, goos, goarch string) (VerifiedMARSReleaseArchive, error) {
			h.events = append(h.events, "verify-archive")
			h.archiveVerifierCalls++
			h.archiveTag, h.archiveCommit = tag, commit
			require.Equal(h.t, h.fixture.bodies[h.fixture.asset(h.fixture.archiveName).ID], archive)
			require.True(h.t, checksums.matchesIdentity(tag, commit))
			if h.archiveVerifierErr != nil {
				return VerifiedMARSReleaseArchive{}, h.archiveVerifierErr
			}
			return VerifiedMARSReleaseArchive{binary: h.candidate}, nil
		},
	}
}

func (h *signedDownloadHarness) roundTrip(req *http.Request) (*http.Response, error) {
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	default:
	}
	path := req.URL.Path
	h.events = append(h.events, path)
	if h.checkRequest != nil {
		h.checkRequest(req)
	}
	switch {
	case path == signedLatestPath() || strings.HasPrefix(path, "/repos/greaveselliott/MARS/releases/tags/"):
		h.latestCalls++
		if h.metadataResponse != nil {
			return h.metadataResponse(req), nil
		}
		return signedDownloadJSONResponse(h.t, req, h.release), nil
	case path == signedReleaseIDPath(h.release.ID):
		final := h.release
		if h.finalRelease != nil {
			final = *h.finalRelease
		}
		return signedDownloadJSONResponse(h.t, req, final), nil
	case path == signedRefPath(h.release.TagName):
		object := h.refObjects[min(h.refCalls, len(h.refObjects)-1)]
		h.refCalls++
		return signedDownloadJSONResponse(h.t, req, githubGitRef{Ref: "refs/tags/" + h.release.TagName, Object: object}), nil
	case strings.HasPrefix(path, "/repos/greaveselliott/MARS/git/tags/"):
		sha := strings.TrimPrefix(path, "/repos/greaveselliott/MARS/git/tags/")
		object, ok := h.annotatedObjects[sha]
		require.True(h.t, ok, "unexpected annotated tag object %s", sha)
		h.annotatedCalls[sha]++
		return signedDownloadJSONResponse(h.t, req, githubTag{SHA: sha, Object: object}), nil
	case strings.HasPrefix(path, "/repos/greaveselliott/MARS/releases/assets/"):
		idText := strings.TrimPrefix(path, "/repos/greaveselliott/MARS/releases/assets/")
		id, err := strconv.ParseInt(idText, 10, 64)
		require.NoError(h.t, err)
		body, ok := h.fixture.bodies[id]
		require.True(h.t, ok, "unexpected asset id %d", id)
		if responder := h.assetResponse[id]; responder != nil {
			return responder(req, body), nil
		}
		return signedDownloadResponse(req, http.StatusOK, body), nil
	default:
		return nil, errors.New("unexpected signed-download request")
	}
}

func cloneGitHubSignedRelease(release githubSignedRelease) githubSignedRelease {
	cloned := release
	cloned.Assets = append([]githubSignedAsset(nil), release.Assets...)
	return cloned
}

func signedDownloadJSONResponse(t *testing.T, req *http.Request, value any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	response := signedDownloadResponse(req, http.StatusOK, raw)
	response.Header.Set("Content-Type", "application/json")
	return response
}

func signedDownloadResponse(req *http.Request, status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Status:        strconv.Itoa(status),
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}

func signedLatestPath() string {
	return "/repos/greaveselliott/MARS/releases/latest"
}

func signedReleaseIDPath(id int64) string {
	return "/repos/greaveselliott/MARS/releases/" + strconv.FormatInt(id, 10)
}

func signedRefPath(tag string) string {
	return "/repos/greaveselliott/MARS/git/ref/tags/" + tag
}
