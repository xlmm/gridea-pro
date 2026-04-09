package domain

import "context"

// AIUsage 内置 Key 的本地调用次数计数
// 仅用于限制使用内置免费 Key 的用户，使用自己 Key 的用户不受影响
type AIUsage struct {
	Date        string `json:"date"`        // YYYY-MM-DD（按本地时间）
	DailyCount  int    `json:"dailyCount"`  // 当日已用次数
	Minute      string `json:"minute"`      // YYYY-MM-DD HH:MM
	MinuteCount int    `json:"minuteCount"` // 当前分钟已用次数
	Sig         string `json:"sig"`         // HMAC-SHA256 签名，防止用户编辑文件直接重置
}

// AIUsageRepository 调用计数存储接口
type AIUsageRepository interface {
	GetAIUsage(ctx context.Context) (AIUsage, error)
	SaveAIUsage(ctx context.Context, usage AIUsage) error
}
