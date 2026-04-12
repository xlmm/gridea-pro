package oauth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ensureGitHubRepo 确保用户的 GitHub Pages 仓库（username.github.io）存在
// 已存在则跳过（无论内容是否为空），不存在则创建公开仓库并自动初始化 README
func ensureGitHubRepo(client *http.Client, token, username string) (string, error) {
	repoName := strings.ToLower(username) + ".github.io"

	// 1. 检查仓库是否存在
	checkURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", username, repoName)
	req, _ := http.NewRequest(http.MethodGet, checkURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Gridea-Pro")

	resp, err := client.Do(req)
	if err != nil {
		return repoName, fmt.Errorf("check repo failed: %w", err)
	}
	resp.Body.Close()

	// 已存在：直接返回，绝不覆盖
	if resp.StatusCode == http.StatusOK {
		return repoName, nil
	}

	// 其他非 404 错误：不创建，避免误操作
	if resp.StatusCode != http.StatusNotFound {
		return repoName, fmt.Errorf("unexpected status when checking repo: %d", resp.StatusCode)
	}

	// 2. 404：创建新仓库
	payload := map[string]interface{}{
		"name":        repoName,
		"description": "My blog powered by Gridea Pro",
		"private":     false,
		"auto_init":   true,
	}
	body, _ := json.Marshal(payload)
	createReq, _ := http.NewRequest(http.MethodPost, "https://api.github.com/user/repos", bytes.NewReader(body))
	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Accept", "application/vnd.github+json")
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("User-Agent", "Gridea-Pro")

	createResp, err := client.Do(createReq)
	if err != nil {
		return repoName, fmt.Errorf("create repo failed: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated {
		return repoName, fmt.Errorf("create repo failed with status %d", createResp.StatusCode)
	}
	return repoName, nil
}

// ensureGiteeRepo 确保用户的 Gitee Pages 仓库（username）存在
func ensureGiteeRepo(client *http.Client, token, username string) (string, error) {
	repoName := strings.ToLower(username)

	// 1. 检查仓库是否存在
	checkURL := fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s?access_token=%s", username, repoName, token)
	req, _ := http.NewRequest(http.MethodGet, checkURL, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Gridea-Pro")

	resp, err := client.Do(req)
	if err != nil {
		return repoName, fmt.Errorf("check repo failed: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return repoName, nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return repoName, fmt.Errorf("unexpected status when checking repo: %d", resp.StatusCode)
	}

	// 2. 创建新仓库
	payload := map[string]interface{}{
		"access_token": token,
		"name":         repoName,
		"description":  "My blog powered by Gridea Pro",
		"private":      false,
		"auto_init":    true,
	}
	body, _ := json.Marshal(payload)
	createReq, _ := http.NewRequest(http.MethodPost, "https://gitee.com/api/v5/user/repos", bytes.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Accept", "application/json")
	createReq.Header.Set("User-Agent", "Gridea-Pro")

	createResp, err := client.Do(createReq)
	if err != nil {
		return repoName, fmt.Errorf("create repo failed: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusCreated && createResp.StatusCode != http.StatusOK {
		return repoName, fmt.Errorf("create repo failed with status %d", createResp.StatusCode)
	}
	return repoName, nil
}
