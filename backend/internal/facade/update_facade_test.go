package facade

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsTransientDownloadErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"500", &httpStatusError{code: 500}, true},
		{"502", &httpStatusError{code: 502}, true},
		{"408", &httpStatusError{code: 408}, true},
		{"429", &httpStatusError{code: 429}, true},
		{"404", &httpStatusError{code: 404}, false},
		{"401", &httpStatusError{code: 401}, false},
		{"unexpected_eof", io.ErrUnexpectedEOF, true},
		{"connection_reset", errors.New("read: connection reset by peer"), true},
		{"broken_pipe", errors.New("write: broken pipe"), true},
		{"no_such_host", errors.New("lookup evil.local: no such host"), true},
		{"unrelated", errors.New("disk full"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTransientDownloadErr(tt.err)
			if got != tt.want {
				t.Errorf("isTransientDownloadErr(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// 500 服务器首两次 503，第三次 200 —— doDownload 应在第 3 次尝试拿到成功。
func TestDoDownload_RetriesOnTransient500(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			http.Error(w, "try again", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Length", "4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OKAY"))
	}))
	defer srv.Close()

	f := &UpdateFacade{httpClient: &http.Client{Timeout: 2 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		f.doDownload(ctx, srv.URL+"/asset.zip", "asset.zip", 4)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("doDownload hung")
	}

	if got := hits.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
	f.mu.Lock()
	ready := f.readyPath
	f.mu.Unlock()
	if ready == "" {
		t.Error("expected readyPath after successful retry")
	}
}

// 4xx 非重试错误（404）应立即放弃，不再发起第二次请求。
func TestDoDownload_NoRetryOn404(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	f := &UpdateFacade{httpClient: &http.Client{Timeout: 2 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	f.doDownload(ctx, srv.URL+"/asset.zip", "asset.zip", 4)

	if got := hits.Load(); got != 1 {
		t.Errorf("4xx should not retry, got %d hits", got)
	}
}

// 用户取消（ctx.Cancel）应立即终止，既不重试也不再发 HTTP 请求。
func TestDoDownload_CancelStopsRetry(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	f := &UpdateFacade{httpClient: &http.Client{Timeout: 2 * time.Second}}
	ctx, cancel := context.WithCancel(context.Background())

	// 在第一次 503 之后立刻取消
	go func() {
		for hits.Load() < 1 {
			time.Sleep(10 * time.Millisecond)
		}
		cancel()
	}()

	done := make(chan struct{})
	go func() {
		f.doDownload(ctx, srv.URL+"/asset.zip", "asset.zip", 4)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("doDownload did not stop after cancel")
	}

	// 应该不会完成 3 次重试（要么 1 次要么极小数，远小于 3）
	if got := hits.Load(); got >= 3 {
		t.Errorf("cancel should have stopped retries earlier, got %d hits", got)
	}
}

// 确保 net 包使用不退化为 unused
var _ = net.Listen
