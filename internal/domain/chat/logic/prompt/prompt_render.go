package prompt

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

const (
	templateDir    = "templates/"
	templateSuffix = ".tmpl"
)

// TemplateRenderer Prompt模板渲染服务
type TemplateRenderer struct {
	cache map[string]*template.Template // key: templatePath, value: *template.Template
}

// NewPromptTemplateLogicImpl 创建Prompt模板服务实例
func NewPromptTemplateLogicImpl() *TemplateRenderer {
	cache := make(map[string]*template.Template)

	// 使用 Glob 匹配所有模板文件
	entries, err := fs.Glob(templateFS, templateDir+"*"+templateSuffix)
	if err != nil {
		panic(fmt.Errorf("遍历模板目录失败: %w", err))
	}

	for _, path := range entries {
		content, err := templateFS.ReadFile(path)
		if err != nil {
			panic(fmt.Errorf("读取模板文件 %s 失败: %w", path, err))
		}

		tmpl, err := template.New(path).Option("missingkey=zero").Parse(string(content))
		if err != nil {
			panic(fmt.Errorf("解析模板文件 %s 失败: %w", path, err))
		}
		cache[path] = tmpl
	}
	return &TemplateRenderer{
		cache: cache,
	}
}

// Render 渲染模板
func (s *TemplateRenderer) Render(templateName string, variables map[string]any) (string, error) {
	templatePath := normalizeTemplatePath(templateName)
	tmpl, ok := s.cache[templatePath]
	if !ok {
		return "", fmt.Errorf("prompt模板不存在: %s", templatePath)
	}

	// 执行渲染（text/template 通过反射自动处理 int/bool/string 等类型）
	var buf strings.Builder
	if err := tmpl.Execute(&buf, normalizeVariables(variables)); err != nil {
		return "", fmt.Errorf("prompt模板渲染失败: %s, err=%w", templatePath, err)
	}
	return buf.String(), nil
}

// normalizeTemplatePath 规范化模板路径
func normalizeTemplatePath(templateName string) string {
	if templateName == "" {
		return ""
	}

	normalized := strings.TrimSpace(templateName)

	// 移除开头的斜杠
	for strings.HasPrefix(normalized, "/") {
		normalized = normalized[1:]
	}

	// 确保以 templateDir 开头
	if !strings.HasPrefix(normalized, templateDir) {
		normalized = templateDir + normalized
	}

	// 确保以 templateSuffix 结尾
	if !strings.HasSuffix(normalized, templateSuffix) {
		normalized = normalized + templateSuffix
	}

	return normalized
}

// normalizeVariables 规范化变量
func normalizeVariables(variables map[string]any) map[string]any {
	normalized := make(map[string]any, len(variables))
	for key, value := range variables {
		if value == nil {
			normalized[key] = ""
			continue
		}
		normalized[key] = value
	}
	return normalized
}
