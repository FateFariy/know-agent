package prompt

import (
	"embed"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

const (
	templateDir    = "templates/"
	templateSuffix = ".tmpl"
)

// TemplateLogicImpl Prompt模板渲染服务实现
type TemplateLogicImpl struct {
	cache sync.Map // key: templatePath, value: *template.Template
}

// NewPromptTemplateLogicImpl 创建Prompt模板服务实例
func NewPromptTemplateLogicImpl() *TemplateLogicImpl {
	return &TemplateLogicImpl{}
}

// Render 渲染模板
func (s *TemplateLogicImpl) Render(templateName string, variables map[string]any) (string, error) {
	templatePath := normalizeTemplatePath(templateName)

	tmpl, err := s.loadTemplate(templatePath)
	if err != nil {
		return "", err
	}

	// 执行渲染（text/template 通过反射自动处理 int/bool/string 等类型）
	var buf strings.Builder
	if err = tmpl.Execute(&buf, normalizeVariables(variables)); err != nil {
		return "", fmt.Errorf("prompt模板渲染失败: %s, err=%w", templatePath, err)
	}
	return buf.String(), nil
}

// loadTemplate 加载（并缓存）已解析的模板
func (s *TemplateLogicImpl) loadTemplate(templatePath string) (*template.Template, error) {
	if cached, ok := s.cache.Load(templatePath); ok {
		if tmpl, ok := cached.(*template.Template); ok {
			return tmpl, nil
		}
	}

	content, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("prompt模板不存在: %s", templatePath)
	}

	// 解析模板；Option("missingkey=zero") 让缺失字段渲染为零值而非报错
	tmpl, err := template.New(templatePath).Option("missingkey=zero").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("prompt模板解析失败: %s, err=%w", templatePath, err)
	}

	actual, _ := s.cache.LoadOrStore(templatePath, tmpl)
	return actual.(*template.Template), nil
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
