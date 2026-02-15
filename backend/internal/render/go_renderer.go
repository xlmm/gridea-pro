package render

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gridea-pro/backend/internal/template"
)

// GoTemplateRenderer Go Templates 渲染器
// 使用 Go 标准库 html/template 实现
type GoTemplateRenderer struct {
	config RenderConfig

	// 基础模板（包含所有 includes）
	baseTmpl *htmltemplate.Template
	baseInit sync.Once
	baseErr  error

	// 模板缓存 (用于 caching full page templates if we wanted,
	// but here we primarily rely on baseTmpl + Clone for partial caching)
	// 为了保持接口一致性，我们还是可以缓存完整 Parse 过的模板
	cache     map[string]*htmltemplate.Template
	cacheLock sync.RWMutex
}

// NewGoTemplateRenderer 创建 Go Templates 渲染器
func NewGoTemplateRenderer(config RenderConfig) *GoTemplateRenderer {
	return &GoTemplateRenderer{
		config: config,
		cache:  make(map[string]*htmltemplate.Template),
	}
}

// Render 实现 ThemeRenderer 接口
func (r *GoTemplateRenderer) Render(templateName string, data *template.TemplateData) (string, error) {
	// 1. 获取完整模板 (Base + Page)
	tmpl, err := r.getTemplate(templateName)
	if err != nil {
		return "", fmt.Errorf("获取模板失败: %w", err)
	}

	// 2. 执行渲染
	var buf bytes.Buffer
	// 注意：html/template 的 Execute 默认执行名为 templateName 的模板，
	// 但实际上 ParseFiles 的主文件名就是 templateName。
	// 这里我们需要确认 Execution 的入口。
	// 当我们 Parse(mainTemplateContent) 时，它的名字取决于 New(name)
	// 所以应该 ExecuteTemplate, 或者如果它就是 root template, Execute.
	if err := tmpl.ExecuteTemplate(&buf, templateName, data); err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}

	return buf.String(), nil
}

// GetEngineType 实现 ThemeRenderer 接口
func (r *GoTemplateRenderer) GetEngineType() string {
	return "gotemplate"
}

// ClearCache 实现 ThemeRenderer 接口
func (r *GoTemplateRenderer) ClearCache() {
	r.cacheLock.Lock()
	defer r.cacheLock.Unlock()

	r.cache = make(map[string]*htmltemplate.Template)

	// Reset base template so it reloads on next request
	r.baseInit = sync.Once{}
	r.baseTmpl = nil
	r.baseErr = nil
}

// getTemplate 获取准备好的模板实例
func (r *GoTemplateRenderer) getTemplate(name string) (*htmltemplate.Template, error) {
	// 1. 检查页面级缓存
	r.cacheLock.RLock()
	if tmpl, ok := r.cache[name]; ok {
		r.cacheLock.RUnlock()
		return tmpl, nil
	}
	r.cacheLock.RUnlock()

	// 2. 确保基础模版已加载 (Includes)
	r.baseInit.Do(func() {
		r.loadBaseTemplate()
	})
	if r.baseErr != nil {
		return nil, r.baseErr
	}

	// 3. 克隆基础模板
	// Clone returns a duplicate of the template, including all associated templates.
	// The actual representation is not copied, but the name space of associated templates is.
	var tmpl *htmltemplate.Template
	var err error

	if r.baseTmpl != nil {
		tmpl, err = r.baseTmpl.Clone()
		if err != nil {
			return nil, fmt.Errorf("克隆基础模板失败: %w", err)
		}
	} else {
		// Fallback if no base template (no includes dir), create empty
		tmpl = htmltemplate.New("root").Funcs(template.TemplateFuncs())
	}

	// 4. 解析当前页面模板
	// 重要的是，New(name) 覆盖了 clone 出来的 root template (如果有的话)
	// 或者我们将页面内容 parse 到这个 clone 出来的 set 中

	// 读取页面文件
	themePath := filepath.Join(r.config.AppDir, "themes", r.config.ThemeName)
	templatesDir := filepath.Join(themePath, "templates")

	mainTemplatePath := filepath.Join(templatesDir, name+".html")
	if _, err := os.Stat(mainTemplatePath); os.IsNotExist(err) {
		mainTemplatePath = filepath.Join(templatesDir, name+".gohtml")
	}

	content, err := os.ReadFile(mainTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("读取页面模板失败 %s: %w", name, err)
	}

	// 解析到 Clone 出来的集合中
	// 注意: 如果内容里没有 {{define "name"}}，它会成为 root template
	// 我们重新命名它为 current name
	_, err = tmpl.New(name).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("解析页面模板失败: %w", err)
	}

	// 5. 存入缓存
	// 这样下次就不用 clone + parse 了，直接 execute
	r.cacheLock.Lock()
	r.cache[name] = tmpl
	r.cacheLock.Unlock()

	return tmpl, nil
}

// loadBaseTemplate 加载 includes 目录下的所有模板作为基础
func (r *GoTemplateRenderer) loadBaseTemplate() {
	themePath := filepath.Join(r.config.AppDir, "themes", r.config.ThemeName)
	includesDir := filepath.Join(themePath, "templates", "includes")

	// 如果 includes 不存在，就跳过
	if _, err := os.Stat(includesDir); os.IsNotExist(err) {
		// No includes, baseTmpl remains nil
		return
	}

	// 创建基础模板容器
	// 这里的 name 不重要，因为会被 clone
	base := htmltemplate.New("base").Funcs(template.TemplateFuncs())

	// 扫描所有 .html 和 .gohtml
	patterns := []string{
		filepath.Join(includesDir, "*.html"),
		filepath.Join(includesDir, "*.gohtml"),
	}

	foundFiles := false
	for _, pattern := range patterns {
		files, _ := filepath.Glob(pattern)
		if len(files) > 0 {
			foundFiles = true
			// ParseGlob 可能会报错如果文件格式不对，我们尝试 ParseFiles
			// 或者手动读取 Parse，以便控制名字
			for _, file := range files {
				content, err := os.ReadFile(file)
				if err != nil {
					continue
				}
				// 自动以文件名(无后缀)作为 template name
				// e.g. includes/head.html -> "head"
				baseName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
				_, err = base.New(baseName).Parse(string(content))
				if err != nil {
					r.baseErr = fmt.Errorf("解析公共模板 %s 失败: %w", file, err)
					return
				}
			}
		}
	}

	if foundFiles {
		r.baseTmpl = base
	}
}
