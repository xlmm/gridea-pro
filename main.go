package main

import (
	"embed"
	"gridea-pro/backend/pkg/boot"
)

// Version 由 CI 通过 -ldflags 注入，本地开发默认 "0.0.0-dev"。
// 必须是 var + 字符串字面量：const 会被编译期内联，函数调用初始化会
// 覆盖注入值，两种形式都会让 -X 静默失效。
var Version = "0.0.0-dev"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	boot.Run(assets, Version)
}
