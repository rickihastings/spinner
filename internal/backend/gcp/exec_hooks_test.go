package gcp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsOnGCE_NotOnGCE(t *testing.T) {
	// When not running on GCE, the metadata server is unreachable,
	// so isOnGCE() returns false. This test runs in CI/local where
	// there is no metadata server.
	// We can't easily mock the HTTP call since isOnGCE uses a hardcoded URL,
	// so we test the false path which is the normal case in CI.
	result := isOnGCE()
	assert.False(t, result)
}

func TestIsOnGCE_WithMockServer(t *testing.T) {
	// Save and restore the original function behavior by testing through
	// a helper that accepts a URL.
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantResult bool
	}{
		{
			name: "metadata server returns 200",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Metadata-Flavor") != "Google" {
					w.WriteHeader(http.StatusForbidden)
					return
				}

				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "ok")
			},
			wantResult: true,
		},
		{
			name: "metadata server returns 404",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantResult: false,
		},
		{
			name: "metadata server returns 500",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			result := isOnGCEWithURL(server.URL)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}

func TestCheckGcloudInstalled(t *testing.T) {
	// gcloud is likely not on PATH in CI, but may be in local dev.
	// We just verify the function doesn't panic and returns a meaningful error.
	err := checkGcloudInstalled()
	if err != nil {
		assert.Contains(t, err.Error(), "gcloud CLI not found on PATH")
	}
	// If err is nil, gcloud is installed — that's fine too.
}
