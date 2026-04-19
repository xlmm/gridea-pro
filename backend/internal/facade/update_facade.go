package facade

import (
	"context"
	"encoding/json"
	"fmt"
	"gridea-pro/backend/internal/utils"
	"gridea-pro/backend/internal/version"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// UpdateFacade 处理程序更新检查
type UpdateFacade struct {
	releasesURL string
	httpClient  *http.Client
}

// NewUpdateFacade 创建 UpdateFacade
func NewUpdateFacade() *UpdateFacade {
	return &UpdateFacade{
		releasesURL: "https://api.github.com/repos/Gridea-Pro/gridea-pro/releases/latest",
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// UpdateInfo 更新检查结果
type UpdateInfo struct {
	HasUpdate      bool   `json:"hasUpdate"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	PublishedAt    string `json:"publishedAt"`
	HtmlURL        string `json:"htmlUrl"`
	Body           string `json:"body"`     // 原始 Markdown
	BodyHTML       string `json:"bodyHtml"` // 渲染后的 HTML（用于弹窗展示）
}

// githubRelease GitHub Release API 返回的最小字段集
type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HtmlURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Body        string `json:"body"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

// CheckUpdate 请求 GitHub Releases 接口，返回版本对比结果
func (f *UpdateFacade) CheckUpdate() (*UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.releasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Gridea-Pro/"+version.Version)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 GitHub Releases 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub Releases 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("解析 Releases 响应失败: %w", err)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	info := &UpdateInfo{
		CurrentVersion: version.Version,
		LatestVersion:  latest,
		PublishedAt:    rel.PublishedAt,
		HtmlURL:        rel.HtmlURL,
		Body:           rel.Body,
		BodyHTML:       utils.ToHTMLUnsafe(rel.Body),
		HasUpdate:      !rel.Draft && !rel.Prerelease && compareSemver(latest, version.Version) > 0,
	}
	return info, nil
}

// MockUpdate 返回一份模拟数据，便于调试 UI（不访问网络）
func (f *UpdateFacade) MockUpdate() *UpdateInfo {
	body := "### ✨ 新功能\n\n- 全新更新检查弹窗\n- 支持从 GitHub Releases 拉取最新版本\n\n### 🐞 修复\n\n- 若干细节优化"
	return &UpdateInfo{
		HasUpdate:      true,
		CurrentVersion: version.Version,
		LatestVersion:  "9.9.9",
		PublishedAt:    time.Now().Format(time.RFC3339),
		HtmlURL:        "https://github.com/Gridea-Pro/gridea-pro/releases",
		Body:           body,
		BodyHTML:       utils.ToHTMLUnsafe(body),
	}
}

// compareSemver 比较两个语义化版本号，a > b 返回 1，a < b 返回 -1，相等返回 0。
// 规则：按 . 分段数值比较，段数不等时缺失段视为 0；无法解析的段按 0 处理。
func compareSemver(a, b string) int {
	as := splitVersion(a)
	bs := splitVersion(b)
	n := max(len(as), len(bs))
	for i := range n {
		av := 0
		bv := 0
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

func splitVersion(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			out = append(out, 0)
			continue
		}
		out = append(out, n)
	}
	return out
}
