package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// geminiProvider Google Gemini /v1beta 协议
type geminiProvider struct {
	baseURL string
}

type geminiContentPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string              `json:"role,omitempty"`
	Parts []geminiContentPart `json:"parts"`
}

type geminiGenerateRequest struct {
	Contents         []geminiContent `json:"contents"`
	GenerationConfig struct {
		Temperature     float64 `json:"temperature"`
		MaxOutputTokens int     `json:"maxOutputTokens"`
	} `json:"generationConfig"`
}

type geminiGenerateResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiContentPart `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type geminiModelListResponse struct {
	Models []struct {
		Name string `json:"name"` // 形如 "models/gemini-1.5-pro"
	} `json:"models"`
}

func (p *geminiProvider) Chat(ctx context.Context, req ChatRequest, apiKey string, httpClient *http.Client) (string, error) {
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}

	body := geminiGenerateRequest{
		Contents: []geminiContent{{
			Parts: []geminiContentPart{{Text: req.Prompt}},
		}},
	}
	body.GenerationConfig.Temperature = req.Temperature
	body.GenerationConfig.MaxOutputTokens = req.MaxTokens

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("请求构建失败: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, req.Model, apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("请求创建失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("响应读取失败: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", errors.New("[UPSTREAM_429] 模型服务繁忙")
	}

	var genResp geminiGenerateResponse
	if err := json.Unmarshal(respBytes, &genResp); err != nil {
		return "", fmt.Errorf("响应解析失败: %w", err)
	}
	if genResp.Error != nil {
		return "", fmt.Errorf("API 错误: %s", genResp.Error.Message)
	}
	if len(genResp.Candidates) == 0 || len(genResp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("模型未返回结果")
	}
	var sb bytes.Buffer
	for _, p := range genResp.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}
	return sb.String(), nil
}

func (p *geminiProvider) ListModels(ctx context.Context, apiKey string, httpClient *http.Client) ([]string, error) {
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}

	url := fmt.Sprintf("%s/models?key=%s", p.baseURL, apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var listResp geminiModelListResponse
	if err := json.Unmarshal(respBytes, &listResp); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}
	models := make([]string, 0, len(listResp.Models))
	for _, m := range listResp.Models {
		// "models/gemini-1.5-pro" → "gemini-1.5-pro"
		id := strings.TrimPrefix(m.Name, "models/")
		if id != "" {
			models = append(models, id)
		}
	}
	return filterChatModels(models), nil
}
