package service

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"gridea-pro/backend/internal/domain"
)

// builtInZhipuAPIKeyEncrypted 内置 Zhipu API Key 的 AES-GCM 密文（Base64 编码）
// 加密方式：EncryptKey(plainKey) — 详见下方函数
// 暂未配置内置 Key，用户需在偏好设置 → AI 配置 中填写自己的 Key
const builtInZhipuAPIKeyEncrypted = "XdSkHxsdio1XcIq3Fggd5yKNW7XWhSB2X4s0XYcKZSTuQal3JSmEODaeFAhch49hMmuC8Tf9gAOmmN4VihzANkzYWTYMR859evS5UeY="

const zhipuEndpoint = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
const defaultAIModel = "glm-4-flash"

// 内置 Key 调用频率限制（仅对使用内置免费 Key 的用户生效）
const (
	builtInDailyLimit  = 20 // 每天最多 20 次
	builtInMinuteLimit = 5  // 每分钟最多 5 次
)

// AIService AI 功能服务
type AIService struct {
	repo        domain.AISettingRepository
	settingRepo domain.SettingRepository
	usageRepo   domain.AIUsageRepository
	usageMu     sync.Mutex
}

func NewAIService(repo domain.AISettingRepository, settingRepo domain.SettingRepository, usageRepo domain.AIUsageRepository) *AIService {
	return &AIService{repo: repo, settingRepo: settingRepo, usageRepo: usageRepo}
}

// httpClient 根据当前代理配置返回合适的 HTTP client
func (s *AIService) httpClient(ctx context.Context) *http.Client {
	if s.settingRepo != nil {
		setting, err := s.settingRepo.GetSetting(ctx)
		if err == nil && setting.ProxyEnabled && setting.ProxyURL != "" {
			return newHTTPClient(setting.ProxyURL)
		}
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// deriveEncryptKey 从 App 名称派生加密密钥（16 字节 AES-128）
func deriveEncryptKey() []byte {
	h := sha256.Sum256([]byte("Gridea Pro"))
	return h[:16]
}

// EncryptKey 将明文 API Key 加密为 Base64 密文
// 供开发者配置内置 Key 时使用：将输出值填入 builtInZhipuAPIKeyEncrypted 常量
func EncryptKey(plainKey string) (string, error) {
	key := deriveEncryptKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plainKey), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptBuiltInKey 解密内置 Key，失败或未配置则返回空字符串
func decryptBuiltInKey() string {
	if builtInZhipuAPIKeyEncrypted == "" {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(builtInZhipuAPIKeyEncrypted)
	if err != nil {
		return ""
	}
	key := deriveEncryptKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return ""
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return ""
	}
	return string(plaintext)
}

// getAPIKey 优先使用用户自己的 Key，否则用内置 Key
// 返回值：apiKey, isBuiltIn（是否使用内置 Key）, error
func (s *AIService) getAPIKey(ctx context.Context) (string, bool, error) {
	setting, err := s.repo.GetAISetting(ctx)
	if err == nil && strings.TrimSpace(setting.ZhipuAPIKey) != "" {
		return strings.TrimSpace(setting.ZhipuAPIKey), false, nil
	}
	if builtIn := decryptBuiltInKey(); builtIn != "" {
		return builtIn, true, nil
	}
	return "", false, errors.New("请在「偏好设置 → AI 配置」中设置 Zhipu API Key")
}

// checkBuiltInQuota 检查内置 Key 的调用配额（不增加计数）
// 错误信息使用 [DAILY_LIMIT] / [RATE_LIMIT] 前缀，供前端 i18n 匹配
func (s *AIService) checkBuiltInQuota(ctx context.Context) error {
	s.usageMu.Lock()
	defer s.usageMu.Unlock()

	usage, _ := s.usageRepo.GetAIUsage(ctx)
	now := time.Now()
	today := now.Format("2006-01-02")
	minute := now.Format("2006-01-02 15:04")

	dailyCount := usage.DailyCount
	if usage.Date != today {
		dailyCount = 0
	}
	minuteCount := usage.MinuteCount
	if usage.Minute != minute {
		minuteCount = 0
	}

	if dailyCount >= builtInDailyLimit {
		return fmt.Errorf("[DAILY_LIMIT] 今日免费额度已用完（%d 次/天），请明日再试，或在「偏好设置 → AI 配置」中填入您自己的 API Key", builtInDailyLimit)
	}
	if minuteCount >= builtInMinuteLimit {
		return fmt.Errorf("[RATE_LIMIT] 调用过于频繁，请稍后再试（限制 %d 次/分钟）", builtInMinuteLimit)
	}
	return nil
}

// recordBuiltInUsage 在调用成功后增加内置 Key 计数
func (s *AIService) recordBuiltInUsage(ctx context.Context) {
	s.usageMu.Lock()
	defer s.usageMu.Unlock()

	usage, _ := s.usageRepo.GetAIUsage(ctx)
	now := time.Now()
	today := now.Format("2006-01-02")
	minute := now.Format("2006-01-02 15:04")

	if usage.Date != today {
		usage.Date = today
		usage.DailyCount = 0
	}
	if usage.Minute != minute {
		usage.Minute = minute
		usage.MinuteCount = 0
	}
	usage.DailyCount++
	usage.MinuteCount++
	_ = s.usageRepo.SaveAIUsage(ctx, usage)
}

// getModel 获取使用的模型，未配置则用默认值
func (s *AIService) getModel(ctx context.Context) string {
	setting, err := s.repo.GetAISetting(ctx)
	if err == nil && strings.TrimSpace(setting.Model) != "" {
		return strings.TrimSpace(setting.Model)
	}
	return defaultAIModel
}

// chatRequest OpenAI 兼容的请求结构
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateSlug 根据文章标题用 AI 生成 SEO 友好的英文 Slug
func (s *AIService) GenerateSlug(ctx context.Context, title string) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", errors.New("文章标题不能为空")
	}

	apiKey, isBuiltIn, err := s.getAPIKey(ctx)
	if err != nil {
		return "", err
	}
	// 仅对使用内置 Key 的用户做本地配额检查
	if isBuiltIn {
		if err := s.checkBuiltInQuota(ctx); err != nil {
			return "", err
		}
	}
	model := s.getModel(ctx)

	prompt := fmt.Sprintf(
		"Generate an SEO-friendly English URL slug from the blog title.\n\n"+
			"Goal: Both search engines and human readers should immediately understand "+
			"what the article is about just by looking at the slug. The slug must read "+
			"like a natural English phrase, not a word-for-word translation.\n\n"+
			"Process (think before writing):\n"+
			"1. Identify the SINGLE main idea of the title (one sentence in your head).\n"+
			"2. If the title has a subtitle (after —, ——, :, or 、), treat it as background "+
			"context only — DO NOT translate it word by word. Use it just to disambiguate the main idea.\n"+
			"3. Express that main idea as a short English phrase: subject + action + (optional context).\n"+
			"4. Trim to 4–8 words. NEVER exceed 8 words. Aim for 5–6.\n\n"+
			"Rules:\n"+
			"- HARD LIMIT: 8 words maximum. Count the words before outputting.\n"+
			"- Drop filler words: a, an, the, is, are, that, how, what, something, anything, everyone, every\n"+
			"- Keep short connectors only when they aid clarity: vs, with, for, to, in\n"+
			"- Brand/tech names must be exact and lowercased (e.g. macos, docker, nextjs, gpt-4, wechat, claude-code, gridea)\n"+
			"- Keep version numbers and years when present (e.g. gpt-4, 2026)\n"+
			"- All lowercase, hyphens as separators, no special characters, no trailing hyphen\n"+
			"- NEVER translate emotional/rhetorical phrases literally (e.g. 「每个想写点什么的人」「都值得」「让世界更美好」)\n\n"+
			"Examples:\n"+
			"- 我用 Claude Code 重构了整个项目的代码 → refactor-entire-project-with-claude-code\n"+
			"- Arc 和 Chrome 哪个更适合开发者日常使用？ → arc-vs-chrome-for-developers\n"+
			"- 独立开发者出海第一步：选对收款工具 → indie-developer-global-payment-tools\n"+
			"- The Best Markdown Editors for Developers in 2026 → best-markdown-editors-for-developers-2026\n"+
			"- 如何用 Docker 部署 Next.js 到生产环境 → deploy-nextjs-to-production-with-docker\n"+
			"- 从零搭建一个个人博客系统 → build-personal-blog-system-from-scratch\n"+
			"- 我为什么复活了 Gridea —— 每个想写点什么的人，都值得一个更简单的开始 → why-i-revived-gridea-for-simpler-writing\n"+
			"- ChatGPT 改变了我的工作方式：从效率工具到思考伙伴 → how-chatgpt-changed-my-workflow\n\n"+
			"Output ONLY the slug string, nothing else. No quotes, no explanation.\n\n"+
			"Title: %s",
		title,
	)

	reqBody := chatRequest{
		Model:       model,
		Messages:    []chatMessage{{Role: "user", Content: prompt}},
		Temperature: 0.1,
		MaxTokens:   80,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("请求构建失败: %w", err)
	}

	client := s.httpClient(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, zhipuEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("请求创建失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("响应读取失败: %w", err)
	}

	// 上游 429（请求过于频繁）单独识别
	if resp.StatusCode == http.StatusTooManyRequests {
		return "", errors.New("[UPSTREAM_429] 智谱 API 当前请求过于频繁，请稍后再试")
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("响应解析失败: %w", err)
	}
	if chatResp.Error != nil {
		return "", fmt.Errorf("API 错误: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return "", errors.New("AI 未返回结果")
	}

	raw := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	// 清理：只保留字母、数字、连字符
	var b strings.Builder
	for _, r := range strings.ToLower(raw) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "", errors.New("生成的 Slug 无效，请重试")
	}

	// 调用成功后，记录内置 Key 使用次数
	if isBuiltIn {
		s.recordBuiltInUsage(ctx)
	}
	return result, nil
}
