package route

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

var (
	// sectionCodePattern 匹配 1.2 / 3.4.5 这类章节编号
	sectionCodePattern = regexp.MustCompile(`(\d+(?:\.\d+)+)`)

	// chineseSectionReferencePattern 匹配"第 3 章 / 第三节 / 第 4 小节"
	chineseSectionReferencePattern = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*(章|节|小节)`)

	// stepReferencePattern 匹配"第几步"
	stepReferencePattern = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*步`)

	// ordinalReferencePattern 匹配"第几条/点/项/个"
	ordinalReferencePattern = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*([条点项个])`)

	// quotedTextPattern 匹配引号包裹的标题短语
	quotedTextPattern = regexp.MustCompile(`["“']([^"”']{2,40})["”']`)
)

// structureNavigationConfidenceThreshold 结构导航置信度阈值
const structureNavigationConfidenceThreshold = 0.65

// DocumentRouterImpl 文档问答结构导航锚点解析器
//
// 本服务不承担"图直答/图定位取证"执行分叉，也不使用关键词、短语剥离或正文 contains 打分决定意图。
// 它只做两件确定性的事：
//  1. 把受控 QueryUnderstandingResult 里的结构导航意图翻译成结构导航动作（结构语法，非业务词）。
//  2. 用精确锚点（章节号 / 引号标题精确 / 导航索引命中）定位结构节点，供确定性结构查询和软提示使用。
//
// 所有文档问答统一输出 ExecutionMode.Retrieval，进入统一多通道混合检索；
// 结构导航结果只作为检索上下文和观测信号，不得绕过混合检索。
type DocumentRouterImpl struct {
	querier GraphQuerier      // 结构图谱查询
	indexer NavigationIndexer // 可选：章节索引服务
}

// NewDocumentRouter 创建文档路由器
func NewDocumentRouter(querier GraphQuerier, indexer NavigationIndexer) *DocumentRouterImpl {
	return &DocumentRouterImpl{
		querier: querier,
		indexer: indexer,
	}
}

// Route 文档问答路由主入口
//
// 执行流程：
//  1. 规范化输入（改写问题、子问题列表、拼接路由文本）
//  2. 获取查询理解结果
//  3. 高置信结构导航时走结构树确定性查询
//  4. 明确编号项时作为结构锚点软辅助混合检索
//  5. 其余按普通文档问题处理
func (r *DocumentRouterImpl) Route(ctx context.Context, input *conversation.DocumentRouteInput) (*vo.DocumentNavigationDecision, error) {
	if input == nil {
		return nil, fmt.Errorf("输入为空")
	}

	// 选取改写后的问题，无改写则回退原始问题
	rewrittenQuestion := input.RewrittenQuestion()

	// 从改写结果中提取子问题列表
	subQuestions := input.SubQuestions()

	// 拼接路由文本（原始 + 改写）
	routeText := input.RouteText()

	// 获取意图识别结果
	recognitionResult := input.RecognitionResult

	// 获取主要检索意图
	retrievalIntent := recognitionResult.PrimaryRetrievalIntent()

	// 高置信结构导航：走结构树确定性查询
	structureNavigationAction := recognitionResult.ResolveAction(structureNavigationConfidenceThreshold)
	if structureNavigationAction != "" {
		section := r.resolveSection(ctx, input.DocumentId, input.OriginalQuestion, rewrittenQuestion)
		return r.buildDecision(structureNavigationAction, section, nil, recognitionResult,
			enum.RetrievalIntentStructure, "高置信结构导航走结构树确定性查询，结构结果作为检索上下文和观测信号。"), nil
	}
	extractor := newNavigationExtractor(routeText)

	// 明确编号项（第 N 步 / 第 N 项）：作为结构锚点软辅助混合检索
	itemIndex := extractor.resolveExplicitItemIndex()
	if len(subQuestions) > 1 {
		itemIndex = nil
	}

	hasExplicitStructureAnchor := itemIndex != nil ||
		extractor.hasExplicitSectionAnchor() ||
		recognitionResult.HasSectionAnchor()

	var assistedSection *vo.GraphSection
	if hasExplicitStructureAnchor {
		assistedSection = r.resolveSection(ctx, input.DocumentId, input.OriginalQuestion, rewrittenQuestion)
	}

	action := enum.DocumentNavigationActionFreshTopic
	if itemIndex != nil {
		action = enum.DocumentNavigationActionItemReference
	}

	reason := "普通文档问题走混合检索"
	if assistedSection != nil {
		reason = "结构锚点仅作为软提示辅助混合检索"
	}

	return r.buildDecision(action, assistedSection, itemIndex, recognitionResult, retrievalIntent, reason), nil
}

// ============================================================
// 导航决策构建
// ============================================================

// buildDecision 构建导航决策
func (r *DocumentRouterImpl) buildDecision(action string, section *vo.GraphSection,
	itemIndex *int, recognitionResult *vo.IntentRecognitionResult,
	retrievalIntent enum.RetrievalIntent, reason string) *vo.DocumentNavigationDecision {

	mode := enum.ExecutionModeRetrieval
	decision := &vo.DocumentNavigationDecision{
		NavigationAction:  action,
		ExecutionMode:     mode,
		ExecutionModeName: mode.Name(),
		RetrievalIntent:   retrievalIntent,
	}

	// 构建结构锚点
	displayTitle := section.DisplayTitle()
	if section != nil {
		decision.StructureAnchor = &vo.ConversationStructureAnchor{
			RootSectionCode:   utils.Trim(section.NodeCode),
			RootSectionTitle:  utils.Trim(section.Title),
			TargetSectionHint: utils.Trim(displayTitle),
			StructureNodeId:   section.NodeId,
			CanonicalPath:     section.CanonicalPath,
			ScopeMode:         "SOFT",
		}
	} else {
		decision.StructureAnchor = &vo.ConversationStructureAnchor{
			ScopeMode: "NONE",
		}
	}

	// 构建条目锚点
	if itemIndex != nil {
		decision.ItemAnchor = &vo.ConversationItemAnchor{
			ItemIndex: *itemIndex,
		}
	}

	queryTypeStr := ""
	source := ""
	if recognitionResult != nil {
		queryTypeStr = recognitionResult.QueryType
		source = recognitionResult.Source
	}

	// 构建摘要文本
	var sb strings.Builder
	sb.WriteString("mode=" + mode.Name())
	sb.WriteString("; retrievalIntent=")
	sb.WriteString(utils.BlankToDefault(retrievalIntent, enum.RetrievalIntentGeneral))
	sb.WriteString("; queryType=")
	sb.WriteString(queryTypeStr)
	sb.WriteString("; intentRecognitionSource=")
	sb.WriteString(source)
	sb.WriteString("; reason=")
	sb.WriteString(reason)
	sb.WriteString("; section=")
	sb.WriteString(displayTitle)
	sb.WriteString("; itemIndex=")
	sb.WriteString(fmt.Sprintf("%d", utils.PointerOrDefault(itemIndex, 0)))
	decision.SummaryText = sb.String()

	logx.Infof("文档问答路由完成: mode=RETRIEVAL, action=%s, section='%s', reason='%s'",
		action, displayTitle, reason)

	return decision
}

// ============================================================
// 章节定位
// ============================================================

// resolveSection 章节定位只允许精确锚点：章节号精确、引号标题精确、导航索引命中。
func (r *DocumentRouterImpl) resolveSection(ctx context.Context, documentId int64,
	originalQuestion, rewrittenQuestion string) *vo.GraphSection {

	if documentId == 0 {
		return nil
	}

	// 先尝试用原始问题定位
	currentTurn := r.resolveSectionByQuestion(ctx, documentId, originalQuestion)
	if currentTurn != nil {
		return currentTurn
	}

	// 如果原始问题和改写问题相同，直接返回 nil
	if utils.Trim(originalQuestion) == utils.Trim(rewrittenQuestion) {
		return nil
	}

	// 再尝试用改写问题定位
	return r.resolveSectionByQuestion(ctx, documentId, rewrittenQuestion)
}

// resolveSectionByQuestion 根据问题定位章节
func (r *DocumentRouterImpl) resolveSectionByQuestion(ctx context.Context,
	documentId int64, question string) *vo.GraphSection {

	// 按章节编号定位
	byCode := r.resolveBySectionCode(ctx, documentId, question)
	if byCode != nil {
		return byCode
	}

	// 按引号标题定位
	byQuotedTitle := r.resolveByQuotedTitle(ctx, documentId, question)
	if byQuotedTitle != nil {
		return byQuotedTitle
	}

	// 按导航索引定位
	return r.resolveByNavigationIndex(ctx, documentId, question)
}

// resolveBySectionCode 按章节编号定位
func (r *DocumentRouterImpl) resolveBySectionCode(ctx context.Context, documentId int64,
	question string) *vo.GraphSection {

	normalized := utils.Trim(question)
	for _, code := range sectionCodePattern.FindAllString(normalized, -1) {
		section, err := r.querier.FindSectionByCode(ctx, documentId, code)
		if err == nil && section != nil {
			return section
		}
	}

	return nil
}

// resolveByQuotedTitle 按引号标题定位
func (r *DocumentRouterImpl) resolveByQuotedTitle(ctx context.Context, documentId int64,
	question string) *vo.GraphSection {
	extractor := newNavigationExtractor(question)

	quotedPhrases := extractor.collectQuotedPhrases()
	if len(quotedPhrases) == 0 {
		return nil
	}

	sections, err := r.querier.ListSections(ctx, documentId)
	if err != nil || len(sections) == 0 {
		return nil
	}

	for _, phrase := range quotedPhrases {
		normalizedPhrase := normalizeText(phrase)
		if len(normalizedPhrase) < 2 {
			continue
		}

		for _, section := range sections {
			normalizedTitle := normalizeText(section.Title)
			normalizedDisplay := normalizeText(section.DisplayTitle())
			normalizedPath := normalizeText(section.SectionPath)

			if normalizedTitle == normalizedPhrase ||
				normalizedDisplay == normalizedPhrase ||
				strings.HasSuffix(normalizedPath, normalizedPhrase) {
				return section
			}
		}
	}

	return nil
}

// resolveByNavigationIndex 按导航索引定位
func (r *DocumentRouterImpl) resolveByNavigationIndex(ctx context.Context, documentId int64,
	question string) *vo.GraphSection {

	if r.indexer == nil {
		return nil
	}
	extractor := newNavigationExtractor(question)
	normalized := utils.Trim(question)
	input := &SearchInput{
		DocumentId: documentId,
		Topic:      normalized,
		Facet:      extractor.detectFacet(),
		Question:   normalized,
		TopK:       5,
	}

	hits, err := r.indexer.SearchSections(ctx, input)
	if err != nil || len(hits) == 0 {
		return nil
	}

	graphSection, err := r.querier.FindSectionById(ctx, documentId, hits[0].NodeId)
	if err != nil {
		return nil
	}

	return graphSection
}

// normalizeText 归一化文本用于精确匹配
func normalizeText(text string) string {
	normalized := utils.Trim(text)
	if normalized == "" {
		return ""
	}

	// 清理特殊字符并转小写
	replacer := strings.NewReplacer(
		" ", "", "\t", "", "\n", "",
		">", "", "`", "*", "", "#", "", "_", "", "-", "",
		"，", "", ",", "", "。", "", "；", "", ";", "",
		"：", "", ":", "", "（", "", "）", "",
		"\"", "", "'", "",
	)
	cleaned := replacer.Replace(normalized)
	return strings.ToLower(cleaned)
}
