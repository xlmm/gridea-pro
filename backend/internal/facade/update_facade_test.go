package facade

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestFacadeWith404 返回一个 UpdateFacade，其 releasesURL 指向本地 404 服务，
// 用于模拟"新下载失败"的场景。clientTimeout 可调以避免测试过慢。
func newTestFacadeWith404(t *testing.T) (*UpdateFacade, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no release", http.StatusNotFound)
	}))
	f := &UpdateFacade{
		releasesURL: srv.URL,
		httpClient:  &http.Client{Timeout: 2 * time.Second},
	}
	return f, func() { srv.Close() }
}

// 关键修复：连续两次下载（第一次成功、第二次失败）后，readyPath 不应指向第一次的文件。
func TestStartDownload_ClearsPreviousReadyState(t *testing.T) {
	f, cleanup := newTestFacadeWith404(t)
	defer cleanup()

	// 模拟上一次下载成功后残留在 facade 上的状态
	stalePath := filepath.Join(t.TempDir(), "old-release.zip")
	if err := os.WriteFile(stalePath, []byte("old content"), 0o644); err != nil {
		t.Fatalf("seed stale file: %v", err)
	}
	f.mu.Lock()
	f.readyPath = stalePath
	f.readyAssetName = "old-release.zip"
	f.mu.Unlock()

	// 新一轮 StartDownload —— 这次因为 releasesURL 返回 404 一定会失败
	if err := f.StartDownload(); err != nil {
		t.Fatalf("StartDownload returned sync error: %v", err)
	}

	// 等待后台 goroutine 结束（clearDownloadState 会清空 downloadCancel）
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		running := f.downloadCancel != nil
		f.mu.Unlock()
		if !running {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	f.mu.Lock()
	gotPath := f.readyPath
	gotName := f.readyAssetName
	f.mu.Unlock()

	if gotPath != "" {
		t.Errorf("readyPath should be cleared after failed new download, got %q", gotPath)
	}
	if gotName != "" {
		t.Errorf("readyAssetName should be cleared, got %q", gotName)
	}
	// 旧 zip 应该已经被 StartDownload 同步清理
	if _, err := os.Stat(stalePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("stale file should have been removed, stat err: %v", err)
	}
}

// ApplyUpdate 在新一轮下载失败后应明确报"尚未完成下载"，而不是静默安装旧版。
func TestApplyUpdate_AfterFailedRedownload_ReturnsNotReady(t *testing.T) {
	f, cleanup := newTestFacadeWith404(t)
	defer cleanup()

	stalePath := filepath.Join(t.TempDir(), "old-release.zip")
	_ = os.WriteFile(stalePath, []byte("old"), 0o644)

	f.mu.Lock()
	f.readyPath = stalePath
	f.readyAssetName = "old-release.zip"
	f.mu.Unlock()

	_ = f.StartDownload()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		running := f.downloadCancel != nil
		f.mu.Unlock()
		if !running {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	err := f.ApplyUpdate()
	if err == nil {
		t.Fatal("expected ApplyUpdate to error after failed redownload")
	}
	if err.Error() != "尚未完成下载，无法安装" {
		t.Errorf("expected '尚未完成下载' error, got %q", err.Error())
	}
}
