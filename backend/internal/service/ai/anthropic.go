package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// anthropicProvider Anthropic /v1/messages 协议
type anthropicProvider struct {
	baseURL string
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicChatRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
}

type anthropicChatResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type anthropicModelListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

const anthropicAPIVersion = "2023-06-01"

func (p *anthropicProvider) Chat(ctx context.Context, req ChatRequest, apiKey string, httpClient *http.Client) (string, error) {
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}

	body := anthropicChatRequest{
		Model:       req.Model,
		Messages:    []anthropicMessage{{Role: "user", Content: req.Prompt}},
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("请求构建失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("请求创建失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

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

	var chatResp anthropicChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("响应解析失败: %w", err)
	}
	if chatResp.Error != nil {
		return "", fmt.Errorf("API 错误: %s", chatResp.Error.Message)
	}
	if len(chatResp.Content) == 0 {
		return "", errors.New("模型未返回结果")
	}
	// 拼接所有 text 块
	var sb bytes.Buffer
	for _, c := range chatResp.Content {
		if c.Type == "text" {
			sb.WriteString(c.Text)
		}
	}
	return sb.String(), nil
}

func (p *anthropicProvider) ListModels(ctx context.Context, apiKey string, httpClient *http.Client) ([]string, error) {
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

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
	var listResp anthropicModelListResponse
	if err := json.Unmarshal(respBytes, &listResp); err != nil {
		return nil, fmt.Errorf("响应解析失败: %w", err)
	}
	models := make([]string, 0, len(listResp.Data))
	for _, m := range listResp.Data {
		if m.ID != "" {
			models = append(models, m.ID)
		}
	}
	return filterChatModels(models), nil
}
