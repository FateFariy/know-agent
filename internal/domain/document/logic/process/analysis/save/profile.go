package save

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

const (
	ProfileStatusSuccess = 2
	ProfileSourceAuto    = "auto"
)

// ProfileGeneratePhase 文档画像生成阶段
type ProfileGeneratePhase struct {
	repo adapter.DocumentRepository
}

// NewProfileGeneratePhase 创建文档画像生成阶段
func NewProfileGeneratePhase(repo adapter.DocumentRepository) *ProfileGeneratePhase {
	return &ProfileGeneratePhase{repo: repo}
}

func (p *ProfileGeneratePhase) Name() string {
	return "文档属性阶段"
}

func (p *ProfileGeneratePhase) Execute(ctx context.Context, saveCtx *Context) error {
	if saveCtx == nil || saveCtx.DocumentId == 0 {
		return nil
	}

	// 获取文档信息用于生成画像
	document, err := p.repo.SelectDocumentById(ctx, saveCtx.DocumentId)
	if err != nil || document == nil {
		return nil
	}

	profile := p.buildProfile(document, saveCtx.AnalysisResult, saveCtx.StructureNodes)
	_ = p.repo.SaveProfile(ctx, profile)

	return nil
}

// buildProfile 构建文档画像
func (p *ProfileGeneratePhase) buildProfile(document *entity.Document, analysisResult *aggregate.AnalysisResult, structureNodes []*entity.StructureNode) *entity.DocumentProfile {
	parsedText := ""
	if analysisResult != nil {
		parsedText = analysisResult.ParsedText
	}

	sectionTitles := p.extractSectionTitles(structureNodes)
	supportsItemLookup := p.hasStepOrListItem(structureNodes)
	supportsGraphOutline := len(sectionTitles) >= 2
	graphFriendly := supportsItemLookup || supportsGraphOutline

	documentType := p.resolveDocumentType(sectionTitles, supportsItemLookup, supportsGraphOutline)
	coreTopics := p.buildCoreTopics(document, sectionTitles)
	exampleQuestions := p.buildExampleQuestions(coreTopics)
	summary := p.buildSummary(document, sectionTitles, parsedText)

	return &entity.DocumentProfile{
		DocumentId:           document.ID,
		ProfileVersion:       1,
		DocumentSummary:      summary,
		DocumentType:         documentType,
		CoreTopics:           joinJsonLikeArray(coreTopics),
		ExampleQuestions:     joinJsonLikeArray(exampleQuestions),
		GraphFriendly:        boolToInt(graphFriendly),
		SupportsGraphOutline: boolToInt(supportsGraphOutline),
		SupportsItemLookup:   boolToInt(supportsItemLookup),
		SupportsGraphAssist:  boolToInt(graphFriendly),
		ProfileSource:        ProfileSourceAuto,
		ProfileStatus:        ProfileStatusSuccess,
	}
}

// extractSectionTitles 从结构节点中提取章节标题
func (p *ProfileGeneratePhase) extractSectionTitles(structureNodes []*entity.StructureNode) []string {
	if len(structureNodes) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var titles []string
	for _, node := range structureNodes {
		if node == nil || node.NodeType != vo.NodeTypeSection {
			continue
		}
		title := strings.TrimSpace(node.Title)
		if title == "" {
			continue
		}
		title = p.stripSectionCode(title)
		if _, exists := seen[title]; exists {
			continue
		}
		seen[title] = struct{}{}
		titles = append(titles, title)
		if len(titles) >= 8 {
			break
		}
	}
	return titles
}

// hasStepOrListItem 检查是否有步骤或列表项节点
func (p *ProfileGeneratePhase) hasStepOrListItem(structureNodes []*entity.StructureNode) bool {
	for _, node := range structureNodes {
		if node == nil {
			continue
		}
		if node.NodeType == enum.NodeTypeStep || node.NodeType == enum.NodeTypeListItem {
			return true
		}
	}
	return false
}

// resolveDocumentType 解析文档类型
func (p *ProfileGeneratePhase) resolveDocumentType(sectionTitles []string, supportsItemLookup, supportsGraphOutline bool) string {
	if supportsItemLookup {
		return "structured_items"
	}
	if supportsGraphOutline {
		return "structured_outline"
	}
	if len(sectionTitles) > 0 {
		return "structured_section"
	}
	return "plain_text"
}

// buildCoreTopics 构建核心主题
func (p *ProfileGeneratePhase) buildCoreTopics(document *entity.Document, sectionTitles []string) []string {
	seen := make(map[string]struct{})
	var topics []string

	for _, title := range sectionTitles {
		topic := strings.TrimSpace(title)
		if topic == "" {
			continue
		}
		if _, exists := seen[topic]; exists {
			continue
		}
		seen[topic] = struct{}{}
		topics = append(topics, topic)
		if len(topics) >= 6 {
			return topics
		}
	}

	fileName := p.stripFileExtension(document.DocumentName)
	if fileName != "" {
		if _, exists := seen[fileName]; !exists {
			topics = append(topics, fileName)
		}
	}

	if len(topics) > 6 {
		topics = topics[:6]
	}
	return topics
}

// buildExampleQuestions 构建示例问题
func (p *ProfileGeneratePhase) buildExampleQuestions(coreTopics []string) []string {
	questions := make([]string, 0, len(coreTopics))
	for _, topic := range coreTopics {
		questions = append(questions, topic+"包含哪些内容？")
		if len(questions) >= 6 {
			break
		}
	}
	return questions
}

// buildSummary 构建摘要
func (p *ProfileGeneratePhase) buildSummary(document *entity.Document, sectionTitles []string, parsedText string) string {
	var builder strings.Builder
	docName := utils.BlankToDefault(document.DocumentName, "未命名文档")
	builder.WriteString("文档《" + docName + "》")

	if len(sectionTitles) > 0 {
		titlesToShow := sectionTitles
		if len(titlesToShow) > 4 {
			titlesToShow = titlesToShow[:4]
		}
		builder.WriteString("主要涵盖：" + strings.Join(titlesToShow, "、") + "。")
	}

	excerpt := strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(parsedText, " "))
	if len([]rune(excerpt)) > 180 {
		excerpt = string([]rune(excerpt)[:180])
	}
	if excerpt != "" {
		builder.WriteString("摘要：" + excerpt)
	}

	return strings.TrimSpace(builder.String())
}

// stripSectionCode 去除章节编号
func (p *ProfileGeneratePhase) stripSectionCode(title string) string {
	normalized := strings.TrimSpace(utils.BlankToDefault(title, ""))
	pattern := regexp.MustCompile(`^(第[一二三四五六七八九十百0-9]+[章节条部分]\s*)|(\d+(?:\.\d+)*\s*)`)
	return strings.TrimSpace(pattern.ReplaceAllString(normalized, ""))
}

// stripFileExtension 去除文件扩展名
func (p *ProfileGeneratePhase) stripFileExtension(fileName string) string {
	normalized := strings.TrimSpace(utils.BlankToDefault(fileName, ""))
	if normalized == "" {
		return ""
	}
	ext := filepath.Ext(normalized)
	if ext == "" {
		return normalized
	}
	return normalized[:len(normalized)-len(ext)]
}

// joinJsonLikeArray 将字符串列表连接为 JSON 数组格式
func joinJsonLikeArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		escaped := strings.ReplaceAll(v, "\"", "\\\"")
		quoted = append(quoted, "\""+escaped+"\"")
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

// boolToInt 布尔值转整数
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
