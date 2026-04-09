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

// openAICompatProvider OpenAI 兼容协议适配器
// 适用于：OpenAI / GLM / Deepseek / Kimi / 小米 / Mistral / Groq /
// 豆包 / 通义千问 / OpenRouter / xAI 等
type openAICompatProvider struct {
	baseURL string
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type openAIModelListResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (p *openAICompatProvider) Chat(ctx context.Context, req ChatRequest, apiKey string, httpClient *http.Client) (string, error) {
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}

	body := openAIChatRequest{
		Model:       req.Model,
		Messages:    []openAIChatMessage{{Role: "user", Content: req.Prompt}},
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("请求构建失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("请求创建失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

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

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("响应解析失败: %w", err)
	}
	if chatResp.Error != nil {
		return "", fmt.Errorf("API 错误: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return "", errors.New("模型未返回结果")
	}
	return chatResp.Choices[0].Message.Content, nil
}

func (p *openAICompatProvider) ListModels(ctx context.Context, apiKey string, httpClient *http.Client) ([]string, error) {
	if httpClient == nil {
		httpClient = defaultHTTPClient()
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

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
	var listResp openAIModelListResponse
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
