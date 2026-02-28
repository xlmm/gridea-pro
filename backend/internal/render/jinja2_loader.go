package render

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ============================================================
// 自定义 Pongo2 模板加载器
// ============================================================
//
// 问题背景：
//   Pongo2（Go 实现的 Jinja2 引擎）的 Lexer 严格禁止 {{ }} 和 {% %} 标签内出现换行符，
//   但标准 Jinja2（Python）和 Django 模板允许这种写法。
//   许多 Jinja2 主题中包含如下合法（按标准 Jinja2）但被 Pongo2 拒绝的代码：
//
//     {{ config.domain ~
//        "/path" }}
//
//     {% if condition and
//           other_condition %}
//
//   逐模板手动修复不可持续（主题更新或用户编辑会恢复）。
//
// 解决方案：
//   创建一个包装加载器（SanitizingLoader），在文件读取后、交给 Pongo2 解析之前，
//   自动将 {{ }}, {% %}, {# #} 标签内的多行内容合并为单行。
//   这在引擎层面透明地解决了兼容性问题，对主题开发者完全无感知。

// reTagBlock 匹配 Jinja2 的三种标签块，支持跨行匹配
// (?s) 开启 DOTALL 模式，使 . 匹配包括换行在内的所有字符
// (?U) 开启非贪婪模式（Ungreedy），确保匹配最短可能的块
var reTagBlock = regexp.MustCompile(`(?sU)({{.+}}|{%.+%}|{#.+#})`)

var (
	// Jinja2 loop 变量到 Pongo2 forloop 变量的自动映射
	reLoopIndex0    = regexp.MustCompile(`\bloop\.index0\b`)
	reLoopRevIndex0 = regexp.MustCompile(`\bloop\.revindex0\b`)
	reLoopIndex     = regexp.MustCompile(`\bloop\.index\b`)
	reLoopRevIndex  = regexp.MustCompile(`\bloop\.revindex\b`)
	reLoopFirst     = regexp.MustCompile(`\bloop\.first\b`)
	reLoopLast      = regexp.MustCompile(`\bloop\.last\b`)
)

// sanitizeTemplate 清理模板内容，将标签内的换行符替换为空格
// 这使得标准 Jinja2 的多行标签写法能被 Pongo2 的严格 Lexer 接受
func sanitizeTemplate(content []byte) []byte {
	return reTagBlock.ReplaceAllFunc(content, func(match []byte) []byte {
		// 将标签内部的换行符替换为空格
		cleaned := bytes.ReplaceAll(match, []byte("\n"), []byte(" "))
		// 同时压缩连续空格为单个空格（可读性更好）
		cleaned = bytes.ReplaceAll(cleaned, []byte("  "), []byte(" "))
		return cleaned
	})
}

// SanitizingLoader 是一个包装 Pongo2 文件加载器的自定义加载器
// 它实现了 pongo2.TemplateLoader 接口（Abs + Get）
// 在 Get 方法中，读取模板文件后自动清理标签内的换行符
type SanitizingLoader struct {
	basePath string // 模板根目录绝对路径
}

// NewSanitizingLoader 创建一个新的清理加载器
func NewSanitizingLoader(basePath string) (*SanitizingLoader, error) {
	// 确保路径存在
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, err
	}
	return &SanitizingLoader{basePath: absPath}, nil
}

// Abs 实现 pongo2.TemplateLoader 接口
// Pongo2 の include/extends 路径始终相对于模板根目录解析
func (l *SanitizingLoader) Abs(base, name string) string {
	// 如果 name 已经是绝对路径，直接返回
	if filepath.IsAbs(name) {
		return name
	}

	// 始终从模板根目录解析，这是 Pongo2 的标准行为
	// 例如：{% include "partials/global-seo.html" %} 始终从 templates/ 根目录查找
	return filepath.Join(l.basePath, name)
}

// Get 实现 pongo2.TemplateLoader 接口
// 读取模板文件并自动清理标签内的换行符
func (l *SanitizingLoader) Get(path string) (io.Reader, error) {
	// 安全检查：确保路径在 basePath 范围内
	absPath := path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(l.basePath, path)
	}

	// 规范化路径，防止路径遍历
	absPath = filepath.Clean(absPath)
	if !strings.HasPrefix(absPath, l.basePath) {
		return nil, os.ErrNotExist
	}

	// 读取模板文件
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	// 预处理：清理标签内的换行符
	cleaned := sanitizeTemplate(content)

	// 透明映射 Jinja2 loop 变量到 Pongo2 forloop 变量
	// Pongo2 的 forloop 变量为 PascalCase，而 Jinja2 用户习惯写 loop.index
	// 替换顺序必须严格（匹配长的在前面，如 index0），防止被短规则截断导致残留字符
	// 另外：loop.length 不做映射，因 Pongo2 的 forloop 没有对应属性可以映射
	cleaned = reLoopIndex0.ReplaceAll(cleaned, []byte("forloop.Counter0"))
	cleaned = reLoopRevIndex0.ReplaceAll(cleaned, []byte("forloop.Revcounter0"))
	cleaned = reLoopIndex.ReplaceAll(cleaned, []byte("forloop.Counter"))
	cleaned = reLoopRevIndex.ReplaceAll(cleaned, []byte("forloop.Revcounter"))
	cleaned = reLoopFirst.ReplaceAll(cleaned, []byte("forloop.First"))
	cleaned = reLoopLast.ReplaceAll(cleaned, []byte("forloop.Last"))

	return bytes.NewReader(cleaned), nil
}
