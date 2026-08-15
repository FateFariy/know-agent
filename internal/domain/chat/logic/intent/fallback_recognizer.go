package intent

import (
	"context"
	"regexp"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

var (
	sectionCodePattern             = regexp.MustCompile(`(\d+(?:\.\d+)+)`)                          // 1.2 / 3.4.5
	chineseSectionReferencePattern = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*(章|节|小节)`)       // 第 3 章 / 第三节 / 第 4 小节
	stepReferencePattern           = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*步`)              // "第几步"，用于结构图定位取证
	ordinalReferencePattern        = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*([条点项个])`)       // 第几条/点/项/个
	quotedTextPattern              = regexp.MustCompile(`[“"']([^”"']{2,40})[”"']`)                 // 引号包裹的标题短语
	normalizePattern               = regexp.MustCompile(`[\s>\` + "`" + `*#_\\-，,。；;：:（）()“”\"']+`) // 中文标点符号
	querySplitPattern              = regexp.MustCompile(`[\s、，,；;：:（）()\-的和及与或]+`)                  // 中文分隔符
)

// DeterministicFallbackRecognizer 基于规则的确定性意图识别器
type DeterministicFallbackRecognizer struct{}

func NewDeterministicFallbackRecognizer() *DeterministicFallbackRecognizer {
	return &DeterministicFallbackRecognizer{}
}

// Name 返回识别器名称
func (p *DeterministicFallbackRecognizer) Name() string {
	return "deterministic-fallback"
}

// Recognize 执行确定性意图识别
func (p *DeterministicFallbackRecognizer) Recognize(ctx context.Context, input *RecognitionInput) (*vo.IntentRecognitionResult, error) {
	question := utils.BlankToDefault(input.OriginalQuestion, input.RewrittenQuestion)
	nq := newNavigationQuestion(question)

	strictStructureNavigation := nq.isStrictStructureNavigation()
	outline := nq.isOutlineNavigation()
	structureNavigation := strictStructureNavigation || outline

	anchors := nq.extractSectionAnchors(strictStructureNavigation)
	structureOperations := nq.determineStructureOperations(strictStructureNavigation, outline)

	queryType := enum.QueryTypeDocumentQA
	structureConfidence := 0.55
	if structureNavigation {
		queryType = enum.QueryTypeStructureNavigation
		structureConfidence = 0.86
	}

	channels := []enum.RetrievalIntent{enum.RetrievalIntentGeneral}
	if structureNavigation || len(anchors) > 0 {
		channels = append(channels, enum.RetrievalIntentStructure)
	}

	reasons := []string{"确定性 fallback 仅识别明确结构导航语法，其余按普通文档问答交多通道检索处理。"}

	result := &vo.IntentRecognitionResult{
		QueryType:      queryType,
		Channels:       channels,
		SectionAnchors: anchors,
		Confidence:     structureConfidence,
		Reasons:        reasons,
		Source:         p.Name(),
	}

	if len(structureOperations) > 0 {
		result.StructureNavigationIntent = &vo.StructureNavigationIntent{
			Operations:     structureOperations,
			SectionAnchors: anchors,
			Confidence:     structureConfidence,
			Source:         p.Name(),
		}
	}

	return result, nil
}

// navigationQuestion 封装导航问题，提供结构导航相关的判定方法
type navigationQuestion struct {
	text                string // 原始问题文本
	asksAdjacentSection bool   // 是否询问相邻章节
	locationHints       bool   // 是否询问章节位置
	hasExplicitAnchor   bool   // 是否包含显式锚点
}

func newNavigationQuestion(text string) *navigationQuestion {
	return &navigationQuestion{text: utils.Trim(text)}
}

// isBlank 判断是否为空
func (q *navigationQuestion) isBlank() bool {
	if q == nil {
		return true
	}
	return q.text == ""
}

// isStrictStructureNavigation 判断是否为严格结构导航问题
func (q *navigationQuestion) isStrictStructureNavigation() bool {
	if q.isBlank() {
		return false
	}
	question := q.text

	// 相邻章节意图
	adjacentHints := []string{
		"上一节", "下一节", "前一节", "后一节", "上一章", "下一章",
		"相邻章节", "同一一级章节",
	}
	q.asksAdjacentSection = strutil.ContainsAny(question, adjacentHints)

	// 章节位置意图
	locationHints := []string{"属于哪个章节", "哪个章节", "哪个小节", "哪一节", "哪一章", "章节位置"}
	q.locationHints = strutil.ContainsAny(question, locationHints)

	// 显式锚点
	q.hasExplicitAnchor = sectionCodePattern.MatchString(question) ||
		chineseSectionReferencePattern.MatchString(question) ||
		quotedTextPattern.MatchString(question)

	return q.asksAdjacentSection || (q.locationHints && q.hasExplicitAnchor)
}

// looksOutlineNavigation 判断是否为目录导航问题
func (q *navigationQuestion) isOutlineNavigation() bool {
	if q.isBlank() {
		return false
	}
	question := q.text
	outlineHints := []string{
		"包含哪些章节", "都包含哪些章节", "有哪些章节", "有哪些小节",
		"包含哪些小节", "章节列表", "小节列表", "子章节", "子小节",
		"下级章节", "展开目录", "列出目录",
	}
	return strutil.ContainsAny(question, outlineHints)
}

// determineStructureOperations 确定结构导航操作列表
func (q *navigationQuestion) determineStructureOperations(strictStructureNavigation, outline bool) []enum.StructureNavigationOperation {
	if outline {
		return []enum.StructureNavigationOperation{enum.SectionWithChildren}
	}
	if !strictStructureNavigation {
		return nil
	}

	if q.asksAdjacentSection {
		return []enum.StructureNavigationOperation{enum.SectionWithSiblings}
	}

	if q.locationHints {
		return []enum.StructureNavigationOperation{enum.ParentSection}
	}

	return []enum.StructureNavigationOperation{enum.CurrentSection}
}

// extractSectionAnchors 从文本中提取章节锚点
func (q *navigationQuestion) extractSectionAnchors(includeQuotedText bool) []string {
	if q.isBlank() {
		return nil
	}
	question := q.text
	seen := make(map[string]bool)
	var anchors []string

	processPattern := func(pattern *regexp.Regexp) bool {
		for _, match := range pattern.FindAllString(question, -1) {
			if !seen[match] {
				seen[match] = true
				anchors = append(anchors, match)
			}
			if len(anchors) >= 8 {
				return true
			}
		}
		return false
	}

	// 章节编号（如 1.2.3）
	if processPattern(sectionCodePattern) {
		return anchors
	}
	// 中文章节引用（如 第3章、第三节）
	if processPattern(chineseSectionReferencePattern) {
		return anchors
	}

	// 引号包裹的文本
	if includeQuotedText {
		for _, match := range quotedTextPattern.FindAllStringSubmatch(question, -1) {
			if len(match) >= 2 {
				phrase := utils.Trim(match[1])
				if phrase == "" && !seen[phrase] {
					seen[phrase] = true
					anchors = append(anchors, phrase)
				}
				if len(anchors) >= 8 {
					return anchors
				}
			}
		}
	}

	return anchors
}
