package domain

import "context"

// AIMode AI 模型使用模式
const (
	AIModeBuiltIn = "builtin" // 使用内置免费模型
	AIModeCustom  = "custom"  // 使用用户自己的 API Key
)

// AISetting AI 功能配置
type AISetting struct {
	Mode   string         `json:"mode"`   // "builtin" | "custom"
	Custom AICustomConfig `json:"custom"` // 自定义模式下的厂商/模型/Key
}

// AICustomConfig 自定义模型配置
type AICustomConfig struct {
	Provider string `json:"provider"` // 厂商 ID，如 "openai" / "anthropic" / "glm" 等
	Model    string `json:"model"`    // 模型 ID
	APIKey   string `json:"apiKey"`   // API Key
}

// AISettingRepository AI 配置存储接口
type AISettingRepository interface {
	GetAISetting(ctx context.Context) (AISetting, error)
	SaveAISetting(ctx context.Context, setting AISetting) error
}
