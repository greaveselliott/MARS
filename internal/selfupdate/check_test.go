package selfupdate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	t.Parallel()
	require.Equal(t, VersionEqual, CompareVersions("0.6.0", "v0.6.0"))
	require.Equal(t, VersionBehind, CompareVersions("0.6.0", "0.7.0"))
	require.Equal(t, VersionAhead, CompareVersions("0.8.0", "0.7.0"))
	require.Equal(t, VersionUnknown, CompareVersions("dev", "0.7.0"))
}

func TestLatestRelease_readsGitHubStyleTag(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/releases/latest", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.7.0"}`))
	}))
	defer server.Close()

	version, err := LatestRelease(context.Background(), server.Client(), server.URL+"/releases/latest")
	require.NoError(t, err)
	require.Equal(t, "0.7.0", version)
}
