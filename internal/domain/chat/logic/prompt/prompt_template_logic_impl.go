package prompt

import (
	"embed"
	"fmt"
	"strings"
	"sync"

	"github.com/valyala/fasttemplate"

	"github.com/swiftbit/know-agent/common/utils"
)

//go:embed templates/*.st
var templateFS embed.FS

const (
	templateDir    = "templates/"
	templateSuffix = ".st"
	startDelimiter = "<"
	endDelimiter   = ">"
)

// TemplateLogicImpl Prompt模板渲染服务实现
type TemplateLogicImpl struct {
	cache sync.Map
}

// NewPromptTemplateLogicImpl 创建Prompt模板服务实例
func NewPromptTemplateLogicImpl() *TemplateLogicImpl {
	return &TemplateLogicImpl{}
}

// Render 渲染模板
func (s *TemplateLogicImpl) Render(templateName string, variables map[string]any) (string, error) {
	templatePath := normalizeTemplatePath(templateName)

	// 从缓存中获取或加载模板
	templateContent, err := s.loadTemplate(templatePath)
	if err != nil {
		return "", err
	}

	// 创建模板渲染器
	tmpl := fasttemplate.New(templateContent, startDelimiter, endDelimiter)

	w := &strings.Builder{}
	// 渲染模板
	if _, err = tmpl.Execute(w, normalizeVariables(variables)); err != nil {
		return "", err
	}

	return w.String(), nil
}

// loadTemplate 加载模板内容
func (s *TemplateLogicImpl) loadTemplate(templatePath string) (string, error) {
	// 尝试从缓存获取
	if cached, ok := s.cache.Load(templatePath); ok {
		if content, ok := cached.(string); ok {
			return content, nil
		}
	}

	// 从embed FS读取模板文件
	content, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("prompt模板不存在: %s", templatePath)
	}

	// 缓存模板内容
	s.cache.Store(templatePath, string(content))

	return string(content), nil
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

	// 确保以templateDir开头
	if !strings.HasPrefix(normalized, templateDir) {
		normalized = templateDir + normalized
	}

	// 确保以templateSuffix结尾
	if !strings.HasSuffix(normalized, templateSuffix) {
		normalized = normalized + templateSuffix
	}

	return normalized
}

// normalizeVariables 规范化变量
func normalizeVariables(variables map[string]any) map[string]any {
	normalized := make(map[string]any)

	if len(variables) == 0 {
		return normalized
	}

	for key, value := range variables {
		normalized[key] = utils.Ternary(value == nil, "", value)
	}

	return normalized
}
