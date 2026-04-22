package facade

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func newWhitelistFacade() *UpdateFacade {
	return &UpdateFacade{
		releasesURL: "https://api.github.com/repos/Gridea-Pro/gridea-pro/releases/latest",
		httpClient:  &http.Client{Timeout: 2 * time.Second},
	}
}

func TestIsTrustedDownloadURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"valid_github_release", trustedDownloadPrefix + "v1.0.0/app.zip", true},
		{"different_repo", "https://github.com/other/project/releases/download/v1.0/app.zip", false},
		{"non_github", "https://evil.example.com/releases/download/v1.0/app.zip", false},
		{"http_scheme", "http://github.com/Gridea-Pro/gridea-pro/releases/download/v1/a.zip", false},
		{"prefix_only_no_path", "https://github.com/Gridea-Pro/gridea-pro/releases/download/", true},
		{"look_alike_domain", "https://github.com.evil.com/Gridea-Pro/gridea-pro/releases/download/", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTrustedDownloadURL(tt.url)
			if got != tt.want {
				t.Errorf("isTrustedDownloadURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

// 非白名单 URL 必须在 doDownload 入口就被拒，不能打到网络。
func TestDoDownload_RejectsUntrustedURL(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake binary"))
	}))
	defer srv.Close()

	f := newWhitelistFacade()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// srv.URL 不属于 github.com/Gridea-Pro/gridea-pro/releases/download/ 前缀
	f.doDownload(ctx, srv.URL+"/some-asset.zip", "some-asset.zip", 1024)

	if n := hits.Load(); n != 0 {
		t.Errorf("untrusted URL should not trigger HTTP request, got %d hits", n)
	}
}
