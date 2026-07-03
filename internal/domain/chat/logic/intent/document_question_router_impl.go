package intent

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	vo2 "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
)

var (
	sectionCodePattern             = regexp.MustCompile(`(\d+(?:\.\d+)+)`)                    // 匹配 1.2 / 3.4.5 这类章节编号
	chineseSectionReferencePattern = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*(章|节|小节)`) // 匹配 "第 3 章 / 第三节 / 第 4 小节"
	stepReferencePattern           = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*步`)        // 匹配 "第几步"，偏向结构图定位取证
	ordinalReferencePattern        = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*([条点项个])`) // 匹配 "第几条/点/项/个"
	quotedTextPattern              = regexp.MustCompile(`[“"']([^”"']{2,40})[”"']`)           // 匹配引号包裹的标题短语
	jsonObjectPattern              = regexp.MustCompile(`\{[\s\S]*}`)                         // 从任意文本中抽取 JSON 对象，兼容模型漂移
	normalizePattern               = regexp.MustCompile(`[\s>\` + "`" + `*#_\\-，,。；;：:（）()“”\"']+`)
)

const graphOnlyIntentConfidenceThreshold = 0.75 // 判定 graph_only 的置信度阈值，低置信统一回退 RAG

// 意图分类所需的关键词短语
var (
	adjacencyHints            = []string{"上一节", "下一节", "前一节", "后一节", "上一章", "下一章", "前一章", "后一章", "上章", "下章", "上一个章节", "下一个章节", "属于哪个章节", "章节位置"}
	outlineExplicitHints      = []string{"包含哪些章节", "都包含哪些章节", "有哪些章节", "有哪些小节", "包含哪些小节", "章节列表", "小节列表", "子章节", "子小节", "下级章节", "展开目录", "列出目录"}
	itemHints                 = []string{"哪一步", "哪一项", "第几步", "第几项", "具体步骤", "步骤中的"}
	analyticStrongHints       = []string{"为什么", "原因", "可能原因", "影响", "区别", "对比", "比较", "如何理解", "怎么理解", "说明了什么", "是否意味着", "是否说明", "分析", "解释"}
	analyticWeakRelationHints = []string{"关系", "关联", "联系", "相a关"}

	structuralRelationHints = []string{
		"前后关系", "相邻关系", "上下级关系", "父子关系", "目录关系", "章节关系",
		"所属关系", "位置关系", "顺序关系", "属于哪个章节", "上级章节", "下级章节",
		"同级章节", "父章节", "子章节",
	}

	graphOnlyBlockingAnalyticHints  = []string{"为什么", "原因", "可能原因", "影响", "区别", "对比", "比较", "如何理解", "怎么理解", "说明了什么", "是否意味着", "是否说明", "分析", "解释"}
	graphOnlyContentHints           = []string{"内容", "要求", "规定", "流程", "步骤", "处理", "执行", "怎么做", "讲了什么", "写了什么", "说了什么"}
	graphOnlyDirectionHints         = []string{"前面", "后面", "上面", "下面", "之前", "之后", "此前", "随后", "后续", "接着", "紧接着", "往前", "往后", "前一个", "后一个", "上一个", "下一个", "上一", "下一", "相邻", "前后", "顺序", "位置", "属于", "上级", "父章节", "同级"}
	graphOnlyExplicitAdjacencyHints = []string{"前一个", "后一个", "上一个", "下一个", "相邻", "前后", "顺序", "位置", "属于", "上级", "父章节", "同级"}
	graphOnlyStructureObjectHints   = []string{"章节", "小节", "这章", "这节", "这部分", "这一章", "该章", "本章", "标题", "目录", "部分", "模块", "节点", "条目"}
	graphOnlyOutlineActionHints     = []string{"下面", "下级", "子章节", "子小节", "子项", "展开", "包含哪些", "包括哪些", "有哪些", "列出", "列一下", "组成", "目录"}
	graphOnlyPronounAnchorHints     = []string{"这个", "该", "它", "刚才", "上述", "上面"}
	graphOnlyAdjacencyAnswerHints   = []string{"哪一节", "哪一章", "哪个章节", "哪个小节", "哪个标题", "哪部分", "哪块"}
)

// ============================================================
// 路由内的意图结构体（仅内部使用）
// ============================================================

// graphOnlyIntentDecision 是否进入 GRAPH_ONLY 的判定
type graphOnlyIntentDecision struct {
	matched    bool
	action     string // SECTION_ADJACENCY_LOOKUP / CHILD_SECTION_DESCEND
	confidence float64
	reason     string
	source     string // rule-adjacency-hint / rule-outline-hint / rule-section-code-direction / llm-xxx
}

// questionIntentDecision 文档问题意图分类结果
type questionIntentDecision struct {
	graphOnly       *graphOnlyIntentDecision
	analytic        bool
	outline         bool
	itemLookup      bool
	structureHint   bool
	contentQuestion bool
	confidence      float64
	reason          string
	source          string
}

// DocumentQuestionRouter 文档问题路由：在某个文档内部进行结构/意图判断并生成导航决策
type DocumentQuestionRouter struct {
	chatModel             *logic.ObservedChatModelImpl[*schema.AgenticMessage]
	structureGraphQuerier vo2.StructureGraphQuerier
	navigationIndexSvc    NavigationIndexService // 可空；非 nil 时用于章节索引定位
	promptTemplateLogic   logic.PromptTemplateLogic
}

// NavigationIndexService 章节索引服务接口（可选，与结构图谱并列）
type NavigationIndexService interface {
	SearchSections(ctx context.Context, documentId int64, query, facet, anchorHint string, topK int) ([]*NavigationSectionHit, error)
}

// NavigationSectionHit 索引服务返回的章节命中
type NavigationSectionHit struct {
	NodeId int64
	Score  float64
}

// NewDocumentQuestionRouter 构造函数
func NewDocumentQuestionRouter(
	chatModel *logic.ObservedChatModelImpl[*schema.AgenticMessage],
	structureGraphQuerier vo2.StructureGraphQuerier,
	navigationIndexSvc NavigationIndexService,
	promptTemplateLogic logic.PromptTemplateLogic,
) *DocumentQuestionRouter {
	return &DocumentQuestionRouter{
		chatModel:             chatModel,
		structureGraphQuerier: structureGraphQuerier,
		navigationIndexSvc:    navigationIndexSvc,
		promptTemplateLogic:   promptTemplateLogic,
	}
}

// Route 根据文档 ID、原问题与改写结果进行路由，返回导航决策
func (r *DocumentQuestionRouter) Route(ctx context.Context, documentId int64, originalQuestion string, rewriteResult *vo.QuestionRewriteResult) (*vo.DocumentNavigationDecision, error) {
	rewrittenQuestion := strutil.Trim(originalQuestion)
	if rewriteResult != nil && strutil.IsNotBlank(rewriteResult.RewrittenQuestion) {
		rewrittenQuestion = strutil.Trim(rewriteResult.RewrittenQuestion)
	}
	subQuestions := normalizeSubQuestions(rewriteResult, rewrittenQuestion)
	retrievalPlan := &vo.RetrievalQuestionPlan{
		RetrievalQuestion: rewrittenQuestion,
		SubQuestions:      subQuestions,
	}
	routeText := strutil.Trim(strutil.Trim(originalQuestion) + " " + rewrittenQuestion)

	// 本地规则 + LLM 兜底，完成意图识别
	questionIntent := r.detectQuestionIntent(ctx, routeText, originalQuestion, rewrittenQuestion, subQuestions)

	// 命中 graph_only 且没有多个子问题时，直接走结构图直答
	if questionIntent.graphOnly.matched && len(subQuestions) <= 1 {
		section := r.resolveSection(ctx, documentId, originalQuestion, rewrittenQuestion)
		return r.buildDecision(
			vo.ExecutionModeGraphOnly,
			questionIntent.graphOnly.action,
			section,
			nil,
			retrievalPlan,
			questionIntent.graphOnly.reason,
		), nil
	}

	// 3) 明确的编号项/步骤型问题，走图定位后取证
	itemIndex := resolveExplicitItemIndex(routeText)
	itemLookupMatched := itemIndex != nil || questionIntent.itemLookup
	if itemLookupMatched && !questionIntent.analytic {
		section := r.resolveSection(ctx, documentId, originalQuestion, rewrittenQuestion)
		return r.buildDecision(
			vo.ExecutionModeGraphThenEvidence,
			vo.DocumentNavigationActionItemReference,
			section,
			itemIndex,
			retrievalPlan,
			"编号项或步骤型问题走图定位取证",
		), nil
	}

	// 4) 分析型 / 目录型 / 结构线索型问题：尝试定位章节作为软提示，其余交给混合检索
	var assistedSection *vo2.GraphSection
	needsStructureAssistedRetrieval := questionIntent.analytic || questionIntent.outline || itemIndex != nil || questionIntent.structureHint
	if needsStructureAssistedRetrieval {
		assistedSection = r.resolveSection(ctx, documentId, originalQuestion, rewrittenQuestion)
	}

	action := vo.DocumentNavigationActionFreshTopic
	if itemIndex != nil {
		action = vo.DocumentNavigationActionItemReference
	}
	reason := "普通文档问题走混合检索"
	if assistedSection != nil {
		reason = "结构线索仅作为软提示辅助混合检索"
	}
	return r.buildDecision(
		vo.ExecutionModeRetrieval,
		action,
		assistedSection,
		itemIndex,
		retrievalPlan,
		reason,
	), nil
}

// ============================================================
// 构建决策输出
// ============================================================

func (r *DocumentQuestionRouter) buildDecision(mode vo.ExecutionMode, action string, section *vo2.GraphSection, itemIndex *int, retrievalPlan *vo.RetrievalQuestionPlan, reason string) *vo.DocumentNavigationDecision {
	decision := vo.NewDocumentNavigationDecision()
	decision.ExecutionMode = mode
	decision.NavigationAction = action
	decision.RetrievalPlan = retrievalPlan

	scopeMode := "SOFT"
	if mode == vo.ExecutionModeGraphOnly {
		scopeMode = "GRAPH"
	} else if mode == vo.ExecutionModeRetrieval && section == nil {
		scopeMode = "NONE"
	}

	if section != nil {
		decision.StructureAnchor = &vo.ConversationStructureAnchor{
			AnchorType:        "SECTION",
			AnchorId:          section.NodeId,
			AnchorName:        strutil.Trim(section.Title),
			SectionTitle:      strutil.Trim(section.Title),
			SectionNodeCode:   strutil.Trim(section.NodeCode),
			CanonicalPath:     strutil.Trim(section.CanonicalPath),
			TargetSectionHint: strutil.Trim(section.DisplayTitle()),
			ScopeMode:         scopeMode,
		}
	} else {
		decision.StructureAnchor = &vo.ConversationStructureAnchor{
			AnchorType: "SECTION",
			ScopeMode:  scopeMode,
		}
	}
	if itemIndex != nil {
		decision.ItemAnchor = &vo.ConversationItemAnchor{
			ItemIndex: *itemIndex,
			ItemText:  "第" + strconv.Itoa(*itemIndex) + "项",
		}
	}

	decision.QueryContextHints = buildQueryHints(retrievalPlan, section, itemIndex)
	decision.SoftSectionHints = buildSoftSectionHints(section)
	decision.SummaryText = buildSummaryText(mode, action, section, itemIndex, reason)

	logx.Infof("文档问答路由完成: documentId=%d, mode=%s, action=%s, section=%q, itemIndex=%+v, reason=%q",
		documentIdSafe(mode), modeName(mode), action, safeDisplayTitle(section), itemIndex, reason)
	return decision
}

// ============================================================
// 问题意图识别（本地规则 + LLM 兜底）
// ============================================================

func (r *DocumentQuestionRouter) detectQuestionIntent(ctx context.Context, routeText, originalQuestion, rewrittenQuestion string, subQuestions []string) *questionIntentDecision {
	normalized := strutil.Trim(routeText)
	if strutil.IsBlank(normalized) {
		return noQuestionIntentDecision("问题为空，跳过路由意图判断。")
	}

	itemLookup := looksExplicitItemQuestion(normalized)
	analytic := looksAnalyticQuestion(normalized)
	outline := asksOutline(normalized)
	contentQuestion := strutil.ContainsAny(normalized, graphOnlyContentHints)
	structureHint := mentionsStructure(normalized) || hasGraphOnlyAnchor(normalized) || outline
	graphOnly := noGraphOnlyIntent("本地规则未命中结构图直答意图。")

	canTryGraphOnlyRules := len(subQuestions) <= 1 && !itemLookup && !contentQuestion && !(analytic && strutil.ContainsAny(normalized, graphOnlyBlockingAnalyticHints))
	if canTryGraphOnlyRules {
		graphOnly = r.detectGraphOnlyIntentByRules(normalized)
	}

	// 本地强命中时直接返回，避免无意义的 LLM 调用
	if graphOnly.matched {
		return &questionIntentDecision{
			graphOnly:       graphOnly,
			analytic:        analytic,
			outline:         outline || graphOnly.action == vo.DocumentNavigationActionChildSectionDescend,
			itemLookup:      itemLookup,
			structureHint:   true,
			contentQuestion: contentQuestion,
			confidence:      graphOnly.confidence,
			reason:          graphOnly.reason,
			source:          graphOnly.source,
		}
	}

	localDecision := &questionIntentDecision{
		graphOnly:       graphOnly,
		analytic:        analytic,
		outline:         outline,
		itemLookup:      itemLookup,
		structureHint:   structureHint,
		contentQuestion: contentQuestion,
		confidence:      0.65,
		reason:          "本地路由意图规则判断完成。",
		source:          "local-rules",
	}

	if !shouldUseLLMQuestionIntent(normalized, subQuestions, localDecision) {
		return localDecision
	}

	return r.classifyQuestionIntentWithModel(ctx, originalQuestion, rewrittenQuestion, normalized, localDecision)
}

// detectGraphOnlyIntentByRules 根据本地规则判断是否允许进入 GRAPH_ONLY
func (r *DocumentQuestionRouter) detectGraphOnlyIntentByRules(question string) *graphOnlyIntentDecision {
	if strutil.ContainsAny(question, adjacencyHints) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionSectionAdjacencyLookup,
			confidence: 1.0,
			reason:     "命中明确相邻章节表达，结构型问题直接走图查询。",
			source:     "rule-adjacency-hint",
		}
	}
	hasSectionCode := sectionCodePattern.MatchString(question)
	hasChineseReference := chineseSectionReferencePattern.MatchString(question)
	hasSectionReference := hasSectionCode || hasChineseReference
	hasExplicitAdjacency := strutil.ContainsAny(question, graphOnlyExplicitAdjacencyHints)
	hasAdjacencyAnswerTarget := strutil.ContainsAny(question, graphOnlyAdjacencyAnswerHints)
	if hasSectionReference && (hasExplicitAdjacency || hasAdjacencyAnswerTarget) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionSectionAdjacencyLookup,
			confidence: 0.92,
			reason:     "命中章节编号与方向词组合，结构相邻关系问题走图查询。",
			source:     "rule-section-code-direction",
		}
	}
	if quotedTextPattern.MatchString(question) && (hasExplicitAdjacency || hasAdjacencyAnswerTarget) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionSectionAdjacencyLookup,
			confidence: 0.9,
			reason:     "命中标题锚点与方向词组合，结构相邻关系问题走图查询。",
			source:     "rule-quoted-title-direction",
		}
	}
	if strutil.ContainsAny(question, graphOnlyPronounAnchorHints) &&
		strutil.ContainsAny(question, graphOnlyDirectionHints) &&
		strutil.ContainsAny(question, graphOnlyAdjacencyAnswerHints) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionSectionAdjacencyLookup,
			confidence: 0.88,
			reason:     "命中指代锚点、方向词和章节答案目标，结构相邻关系问题走图查询。",
			source:     "rule-pronoun-direction-answer",
		}
	}
	if strutil.ContainsAny(question, graphOnlyStructureObjectHints) && strutil.ContainsAny(question, graphOnlyExplicitAdjacencyHints) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionSectionAdjacencyLookup,
			confidence: 0.86,
			reason:     "命中结构对象与方向关系组合，结构相邻关系问题走图查询。",
			source:     "rule-structure-direction",
		}
	}
	if asksOutline(question) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionChildSectionDescend,
			confidence: 1.0,
			reason:     "命中明确章节展开表达，结构型问题直接走图查询。",
			source:     "rule-outline-hint",
		}
	}
	if hasGraphOnlyAnchor(question) && strutil.ContainsAny(question, graphOnlyOutlineActionHints) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionChildSectionDescend,
			confidence: 0.86,
			reason:     "命中章节锚点与目录展开动作，结构型问题直接走图查询。",
			source:     "rule-outline-action",
		}
	}
	return noGraphOnlyIntent("本地规则未命中结构图直答意图。")
}

// shouldUseLLMQuestionIntent 判断是否值得调用 LLM 进行兜底意图判断
func shouldUseLLMQuestionIntent(question string, subQuestions []string, decision *questionIntentDecision) bool {
	if len(subQuestions) > 1 {
		return false
	}
	if decision.itemLookup || decision.contentQuestion {
		return false
	}
	if decision.analytic && strutil.ContainsAny(question, graphOnlyBlockingAnalyticHints) {
		return false
	}
	hasGraphOnlyNavigationCue := strutil.ContainsAny(question, graphOnlyDirectionHints) || strutil.ContainsAny(question, graphOnlyOutlineActionHints)
	if hasGraphOnlyAnchor(question) && hasGraphOnlyNavigationCue {
		return true
	}
	return strutil.ContainsAny(question, analyticWeakRelationHints) && decision.structureHint
}

// ============================================================
// LLM 兜底意图检测
// ============================================================

type llmQuestionIntentPayload struct {
	IntentType    string  `json:"intent_type"`
	Action        string  `json:"action"`
	GraphOnly     bool    `json:"graph_only"`
	Analytic      bool    `json:"analytic"`
	Outline       bool    `json:"outline"`
	ItemLookup    bool    `json:"item_lookup"`
	ContentQA     bool    `json:"content_qa"`
	StructureHint bool    `json:"structure_hint"`
	Confidence    float64 `json:"confidence"`
	Reason        string  `json:"reason"`
}

func (r *DocumentQuestionRouter) classifyQuestionIntentWithModel(ctx context.Context, originalQuestion, rewrittenQuestion, routeText string, localDecision *questionIntentDecision) *questionIntentDecision {
	if r.chatModel == nil || r.promptTemplateLogic == nil {
		return localDecision
	}
	promptText, err := r.promptTemplateLogic.Render(prompt.DocumentGraphOnlyIntent, map[string]any{
		"originalQuestion":  strutil.Trim(originalQuestion),
		"rewrittenQuestion": strutil.Trim(rewrittenQuestion),
		"routeText":         strutil.Trim(routeText),
	})
	if err != nil {
		logx.Errorf("渲染文档路由 LLM 模板失败: question=%q, err=%v", originalQuestion, err)
		return localDecision
	}
	opts := []model.Option{model.WithTemperature(0.0), model.WithTopP(0.1)}
	raw, err := r.chatModel.Generate(ctx, "", promptText, opts...)
	if err != nil {
		logx.Errorf("文档路由 LLM 兜底判断失败，回退本地路由意图: question=%q, err=%v", originalQuestion, err)
		return localDecision
	}
	decision := parseQuestionIntentResult(raw, localDecision)
	logx.Infof("文档路由 LLM 兜底判断完成: %+v, raw=%s", decision, raw)
	return decision
}

// parseQuestionIntentResult 从模型返回文本中抽取 JSON，失败时回退本地决策
func parseQuestionIntentResult(raw string, localDecision *questionIntentDecision) *questionIntentDecision {
	if strutil.IsBlank(raw) {
		return localDecision
	}
	jsonStr := extractJsonObject(raw)
	var payload llmQuestionIntentPayload
	if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
		logx.Errorf("解析文档路由 LLM 输出失败: raw=%q, err=%v", raw, err)
		return localDecision
	}
	confidence := normalizeConfidence(payload.Confidence)
	if confidence < graphOnlyIntentConfidenceThreshold {
		return localDecision
	}
	reason := utils.BlankToDefault(strutil.Trim(payload.Reason), "LLM 判定完成。。")
	action := resolveModelGraphOnlyAction(payload.Action, payload.IntentType)
	modelGraphOnlyAccepted := payload.GraphOnly && action != "" &&
		(strings.EqualFold(payload.IntentType, "ADJACENCY") || strings.EqualFold(payload.IntentType, "OUTLINE"))

	graphOnly := noGraphOnlyIntent("LLM 判定不适合结构图直答: " + reason)
	if modelGraphOnlyAccepted {
		graphOnly = &graphOnlyIntentDecision{
			matched:    true,
			action:     action,
			confidence: confidence,
			reason:     "LLM 兜底判定为结构图直答: " + reason,
			source:     "llm-" + strings.ToUpper(payload.IntentType),
		}
	}

	return &questionIntentDecision{
		graphOnly:       graphOnly,
		analytic:        payload.Analytic || strings.EqualFold(payload.IntentType, "ANALYTIC"),
		outline:         payload.Outline || strings.EqualFold(payload.IntentType, "OUTLINE"),
		itemLookup:      payload.ItemLookup || strings.EqualFold(payload.IntentType, "ITEM_LOOKUP"),
		structureHint:   payload.StructureHint || strings.EqualFold(payload.IntentType, "ADJACENCY") || strings.EqualFold(payload.IntentType, "OUTLINE"),
		contentQuestion: payload.ContentQA || strings.EqualFold(payload.IntentType, "CONTENT_QA"),
		confidence:      confidence,
		reason:          "LLM 兜底路由意图判断: " + reason,
		source:          "llm-" + strings.ToUpper(payload.IntentType),
	}
}

func resolveModelGraphOnlyAction(rawAction, intentType string) string {
	action := strings.ToUpper(strutil.Trim(rawAction))
	if action == vo.DocumentNavigationActionSectionAdjacencyLookup {
		return vo.DocumentNavigationActionSectionAdjacencyLookup
	}
	if action == vo.DocumentNavigationActionChildSectionDescend {
		return vo.DocumentNavigationActionChildSectionDescend
	}
	switch strings.ToUpper(intentType) {
	case "ADJACENCY":
		return vo.DocumentNavigationActionSectionAdjacencyLookup
	case "OUTLINE":
		return vo.DocumentNavigationActionChildSectionDescend
	}
	return ""
}

func extractJsonObject(raw string) string {
	trimmed := strutil.Trim(raw)
	if matched := jsonObjectPattern.FindString(trimmed); matched != "" {
		return matched
	}
	return trimmed
}

func normalizeConfidence(confidence float64) float64 {
	if confidence > 1 {
		return confidence / 100.0
	}
	return max(0, confidence)
}

// ============================================================
// 章节定位
// ============================================================

// resolveSection 依次：章节编号 → 索引服务 → 本地短语打分 → 兜底 bestSection
func (r *DocumentQuestionRouter) resolveSection(ctx context.Context, documentId int64, originalQuestion, rewrittenQuestion string) *vo2.GraphSection {
	if documentId == 0 || r.structureGraphQuerier == nil {
		return nil
	}
	section := r.resolveBySectionCode(ctx, documentId, originalQuestion, rewrittenQuestion)
	if section != nil {
		return section
	}
	section = r.resolveByNavigationIndex(ctx, documentId, originalQuestion, rewrittenQuestion)
	if section != nil {
		return section
	}
	phrases := r.buildSectionPhrases(originalQuestion, rewrittenQuestion)
	section = r.resolveByLocalStructure(ctx, documentId, phrases)
	if section != nil {
		return section
	}
	section, err := r.structureGraphQuerier.FindBestSection(ctx, documentId, rewrittenQuestion, "")
	if err != nil {
		logx.Errorf("FindBestSection 调用失败: documentId=%d, err=%v", documentId, err)
		return nil
	}
	return section
}

// resolveBySectionCode 从问题文本中抽取章节编号（小数编号 / 中文编号）进行定位
func (r *DocumentQuestionRouter) resolveBySectionCode(ctx context.Context, documentId int64, originalQuestion, rewrittenQuestion string) *vo2.GraphSection {
	combined := strutil.Trim(originalQuestion) + " " + strutil.Trim(rewrittenQuestion)

	// 1.2.3 类编号
	for _, code := range sectionCodePattern.FindAllString(combined, -1) {
		section, err := r.structureGraphQuerier.FindSectionByCode(ctx, documentId, code)
		if err == nil && section != nil {
			return section
		}
	}
	// 中文编号引用：第 3 章 / 第三节
	for _, m := range chineseSectionReferencePattern.FindAllStringSubmatch(combined, -1) {
		if len(m) < 2 {
			continue
		}
		parsed := utils.ParseChineseNumber(m[1])
		if parsed <= 0 {
			continue
		}
		code := strconv.Itoa(parsed)
		if section, err := r.structureGraphQuerier.FindSectionByCode(ctx, documentId, code); err == nil && section != nil {
			return section
		}
	}
	return nil
}

// resolveByLocalStructure 在章节列表上用本地关键词打分；得分 >= 45 才返回
func (r *DocumentQuestionRouter) resolveByLocalStructure(ctx context.Context, documentId int64, phrases []string) *vo2.GraphSection {
	if len(phrases) == 0 {
		return nil
	}
	sections, err := r.structureGraphQuerier.ListSections(ctx, documentId)
	if err != nil || len(sections) == 0 {
		return nil
	}

	var bestSection *vo2.GraphSection
	bestScore := 0.0
	for _, s := range sections {
		score := scoreSection(s, phrases)
		if score >= 45 && score > bestScore {
			bestSection = s
			bestScore = score
		}
	}
	return bestSection
}

// resolveByNavigationIndex 可选的章节索引服务，命中时直接用图谱查询定位具体节点
func (r *DocumentQuestionRouter) resolveByNavigationIndex(ctx context.Context, documentId int64, originalQuestion, rewrittenQuestion string) *vo2.GraphSection {
	if r.navigationIndexSvc == nil || r.structureGraphQuerier == nil {
		return nil
	}
	query := utils.BlankToDefault(rewrittenQuestion, originalQuestion)
	facet := detectFacet(query)
	hits, err := r.navigationIndexSvc.SearchSections(ctx, documentId, query, facet, "", 5)
	if err != nil || len(hits) == 0 {
		return nil
	}
	// 此处与 Java 保持一致：索引只提供 nodeId，具体节点仍由图谱服务给出
	sections, listErr := r.structureGraphQuerier.ListSections(ctx, documentId)
	if listErr != nil || len(sections) == 0 {
		return nil
	}
	hit := hits[0]
	for _, s := range sections {
		if s != nil && s.NodeId == hit.NodeId {
			return s
		}
	}
	return nil
}

// ============================================================
// 短语抽取与打分
// ============================================================

// buildSectionPhrases 组装用于章节匹配的短语列表（上限 8）
func (r *DocumentQuestionRouter) buildSectionPhrases(originalQuestion, rewrittenQuestion string) []string {
	seen := make(map[string]struct{})
	phrases := make([]string, 0, 8)
	addIfAbsent := func(p string) {
		cleaned := cleanPhrase(p)
		_, ok := seen[cleaned]
		if strutil.IsNotBlank(cleaned) && !ok {
			seen[cleaned] = struct{}{}
			phrases = append(phrases, cleaned)
		}
	}

	addIfAbsent(originalQuestion)
	addIfAbsent(rewrittenQuestion)
	for _, q := range extractQuotedPhrases(originalQuestion) {
		addIfAbsent(q)
	}
	for _, q := range extractQuotedPhrases(rewrittenQuestion) {
		addIfAbsent(q)
	}
	combined := strutil.Trim(originalQuestion) + " " + strutil.Trim(rewrittenQuestion)
	for _, marker := range adjacencyHints {
		addIfAbsent(textBeforeMarker(combined, marker))
	}
	for _, marker := range outlineExplicitHints {
		addIfAbsent(textBeforeMarker(combined, marker))
	}
	for _, step := range stepReferencePattern.FindAllString(combined, -1) {
		addIfAbsent(textBeforeMarker(combined, step))
	}

	// 长度过滤（与 Java 的 limit 一致）
	filtered := phrases[:0]
	for _, p := range phrases {
		if len(normalizeForSection(p)) >= 2 {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) > 8 {
		filtered = filtered[:8]
	}
	return filtered
}

// scoreSection 对单个章节按标题/路径/锚/正文叠加打分
func scoreSection(section *vo2.GraphSection, phrases []string) float64 {
	if section == nil || len(phrases) == 0 {
		return 0
	}
	title := normalizeForSection(section.Title)
	path := normalizeForSection(section.SectionPath)
	anchor := normalizeForSection(section.AnchorText)
	content := normalizeForSection(section.ContentText)

	var best float64
	for _, phrase := range phrases {
		normalized := normalizeForSection(phrase)
		normalizedLen := float64(utf8.RuneCountInString(normalized))
		if normalizedLen < 2 {
			continue
		}
		if strings.Contains(path, normalized) {
			best = max(best, 100.0+normalizedLen)
		}
		if strings.Contains(title, normalized) {
			best = max(best, 90.0+normalizedLen)
		}
		if strings.Contains(anchor, normalized) {
			best = max(best, 80.0+normalizedLen)
		}
		if strings.Contains(content, normalized) {
			best = max(best, 45.0+min(normalizedLen, 20.0))
		}
	}
	return best
}

// ============================================================
// 小工具函数
// ============================================================

func looksExplicitItemQuestion(question string) bool {
	if strutil.ContainsAny(question, itemHints) {
		return true
	}
	return resolveExplicitItemIndex(question) != nil
}

func looksAnalyticQuestion(question string) bool {
	if strutil.ContainsAny(question, analyticStrongHints) {
		return true
	}
	if !strutil.ContainsAny(question, analyticWeakRelationHints) {
		return false
	}
	return !looksStructuralRelationQuestion(question)
}

func looksStructuralRelationQuestion(question string) bool {
	if strutil.ContainsAny(question, structuralRelationHints) {
		return true
	}
	if !hasGraphOnlyAnchor(question) {
		return false
	}
	return strutil.ContainsAny(question, graphOnlyExplicitAdjacencyHints) || strutil.ContainsAny(question, graphOnlyDirectionHints)
}

func mentionsStructure(question string) bool {
	return strutil.ContainsAny(question, []string{"章节", "小节", "条目", "步骤", "项"}) ||
		quotedTextPattern.MatchString(question) ||
		sectionCodePattern.MatchString(question)
}

func hasGraphOnlyAnchor(question string) bool {
	return sectionCodePattern.MatchString(question) ||
		chineseSectionReferencePattern.MatchString(question) ||
		quotedTextPattern.MatchString(question) ||
		strutil.ContainsAny(question, graphOnlyStructureObjectHints) ||
		strutil.ContainsAny(question, graphOnlyPronounAnchorHints)
}

func asksOutline(question string) bool {
	if strutil.IsBlank(question) {
		return false
	}
	if strutil.ContainsAny(question, graphOnlyContentHints) {
		return false
	}
	return strutil.ContainsAny(question, outlineExplicitHints) ||
		(hasGraphOnlyAnchor(question) && strutil.ContainsAny(question, graphOnlyOutlineActionHints))
}

func normalizeForSection(text string) string {
	if strutil.IsBlank(text) {
		return ""
	}
	cleaned := normalizePattern.ReplaceAllString(text, "")
	return strings.ToLower(cleaned)
}

func cleanPhrase(text string) string {
	cleaned := strutil.Trim(text)
	if strutil.IsBlank(cleaned) {
		return ""
	}
	noise := []string{
		"刚才说的", "请问", "帮我", "这个", "那个", "所属的具体章节", "所属章节",
		"具体章节", "章节", "小节", "目录", "上一节", "下一节", "分别是什么",
		"是什么", "有哪些", "都有哪些", "包含哪些", "中的", "里面的", "里的", "中",
		"“", "”", "?", "？",
	}
	for _, n := range noise {
		cleaned = strings.ReplaceAll(cleaned, n, "")
	}
	return strutil.Trim(cleaned)
}

func extractQuotedPhrases(text string) []string {
	matches := quotedTextPattern.FindAllStringSubmatch(strutil.Trim(text), -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			out = append(out, strutil.Trim(m[1]))
		}
	}
	return out
}

func textBeforeMarker(text, marker string) string {
	if strutil.IsBlank(text) || strutil.IsBlank(marker) {
		return ""
	}
	idx := strings.Index(text, marker)
	if idx <= 0 {
		return ""
	}
	return strutil.Trim(text[:idx])
}

func normalizeSubQuestions(rewriteResult *vo.QuestionRewriteResult, fallback string) []string {
	if rewriteResult == nil || len(rewriteResult.SubQuestions) == 0 {
		return []string{strutil.Trim(fallback)}
	}
	out := make([]string, 0, len(rewriteResult.SubQuestions))
	seen := make(map[string]struct{})
	for _, q := range rewriteResult.SubQuestions {
		trimmed := strutil.Trim(q)
		if strutil.IsNotBlank(trimmed) {
			if _, ok := seen[trimmed]; !ok {
				seen[trimmed] = struct{}{}
				out = append(out, trimmed)
			}
		}
	}
	if len(out) == 0 {
		return []string{strutil.Trim(fallback)}
	}
	return out
}

func resolveExplicitItemIndex(question string) *int {
	for _, m := range stepReferencePattern.FindAllStringSubmatch(question, -1) {
		if len(m) >= 2 {
			parsed := utils.ParseChineseNumber(m[1])
			if parsed > 0 {
				return utils.Pointer(parsed)
			}
		}
	}
	for _, m := range ordinalReferencePattern.FindAllStringSubmatch(question, -1) {
		if len(m) >= 2 {
			parsed := utils.ParseChineseNumber(m[1])
			if parsed > 0 {
				return utils.Pointer(parsed)
			}
		}
	}
	return nil
}

func detectFacet(question string) string {
	if strutil.ContainsAny(question, adjacencyHints) {
		return "章节位置"
	}
	if asksOutline(question) {
		return "章节"
	}
	if strutil.ContainsAny(question, itemHints) {
		return "步骤"
	}
	return ""
}

// ============================================================
// 构造决策输出辅助
// ============================================================

func buildQueryHints(retrievalPlan *vo.RetrievalQuestionPlan, section *vo2.GraphSection, itemIndex *int) []string {
	seen := make(map[string]struct{})
	hints := make([]string, 0, 10)
	add := func(h string) {
		h = strutil.Trim(h)
		if strutil.IsBlank(h) {
			return
		}
		if _, ok := seen[h]; ok {
			return
		}
		seen[h] = struct{}{}
		hints = append(hints, h)
	}

	if retrievalPlan != nil {
		if strutil.IsNotBlank(retrievalPlan.RetrievalQuestion) {
			for _, term := range splitRoughTerms(retrievalPlan.RetrievalQuestion) {
				add(term)
			}
			add(retrievalPlan.RetrievalQuestion)
		}
		for _, sub := range retrievalPlan.SubQuestions {
			add(sub)
		}
	}
	if section != nil {
		add(section.DisplayTitle())
		add(strutil.Trim(section.AnchorText))
		add(strutil.Trim(section.NodeCode))
	}
	if itemIndex != nil {
		add("第" + strconv.Itoa(*itemIndex) + "步")
		add("第" + strconv.Itoa(*itemIndex) + "项")
	}
	if len(hints) > 10 {
		hints = hints[:10]
	}
	return hints
}

// splitRoughTerms 将问题按常见分隔符切分为关键词
func splitRoughTerms(text string) []string {
	cleaned := strutil.Trim(text)
	if strutil.IsBlank(cleaned) {
		return nil
	}
	seps := []string{" ", "、", "，", ",", ";", "；", ":", "：", "的", "和", "及", "与", "或"}
	working := cleaned
	for _, sep := range seps {
		working = strings.ReplaceAll(working, sep, "|")
	}
	raw := strings.Split(working, "|")
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{})
	for _, r := range raw {
		t := strutil.Trim(r)
		if strutil.IsBlank(t) || len(t) < 2 {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	limit := 6
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func buildSoftSectionHints(section *vo2.GraphSection) []string {
	if section == nil {
		return []string{}
	}
	title := strutil.Trim(section.DisplayTitle())
	if strutil.IsBlank(title) {
		return []string{}
	}
	return []string{title}
}

func buildSummaryText(mode vo.ExecutionMode, action string, section *vo2.GraphSection, itemIndex *int, reason string) string {
	sectionTitle := safeDisplayTitle(section)
	itemIndexStr := ""
	if itemIndex != nil {
		itemIndexStr = strconv.Itoa(*itemIndex)
	}
	return "mode=" + modeName(mode) + "; action=" + action + "; section=" + sectionTitle + "; itemIndex=" + itemIndexStr + "; reason=" + strutil.Trim(reason)
}

func modeName(mode vo.ExecutionMode) string {
	if mode == nil {
		return "retrieval"
	}
	return mode.Name()
}

func safeDisplayTitle(section *vo2.GraphSection) string {
	if section == nil {
		return ""
	}
	return section.DisplayTitle()
}

func documentIdSafe(_ vo.ExecutionMode) int64 {
	// 为日志参数占位，与 Java 的日志风格对齐
	return -1
}

// ============================================================
// 空意图决策构造器
// ============================================================

func noGraphOnlyIntent(reason string) *graphOnlyIntentDecision {
	return &graphOnlyIntentDecision{
		reason: reason,
		source: "none",
	}
}

func noQuestionIntentDecision(reason string) *questionIntentDecision {
	return &questionIntentDecision{
		graphOnly: noGraphOnlyIntent(reason),
		reason:    reason,
		source:    "none",
	}
}
