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
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/graph"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	vo2 "github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

var (
	sectionCodePattern             = regexp.MustCompile(`(\d+(?:\.\d+)+)`)                          // 1.2 / 3.4.5
	chineseSectionReferencePattern = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*(章|节|小节)`)       // 第 3 章 / 第三节 / 第 4 小节
	stepReferencePattern           = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*步`)              // "第几步"，用于结构图定位取证
	ordinalReferencePattern        = regexp.MustCompile(`第\s*([0-9一二三四五六七八九十百]+)\s*([条点项个])`)       // 第几条/点/项/个
	quotedTextPattern              = regexp.MustCompile(`[“"']([^”"']{2,40})[”"']`)                 // 引号包裹的标题短语
	jsonObjectPattern              = regexp.MustCompile(`\{[\s\S]*}`)                               // 从任意文本中抽取 JSON 对象，兼容模型漂移
	normalizePattern               = regexp.MustCompile(`[\s>\` + "`" + `*#_\\-，,。；;：:（）()“”\"']+`) // 中文标点符号
	querySplitPattern              = regexp.MustCompile(`[\s、，,；;：:（）()\-的和及与或]+`)                  // 中文分隔符
)

// graphOnlyThreshold 判断进入 GRAPH_ONLY 模式的置信度阈值，低于该值统一回退到 RAG
const graphOnlyThreshold = 0.75

// 关键词提示短语：按 "判定意图 → 章节定位 → 混合检索" 的顺序组织
var (
	// adjacencyHints 明确表达"相邻章节"的意图（上一节 / 下一章 ……）
	adjacencyHints = []string{"上一节", "下一节", "前一节", "后一节", "上一章", "下一章", "前一章", "后一章", "上章", "下章", "上一个章节", "下一个章节", "属于哪个章节", "章节位置"}

	// outlineExplicitHints 明确表达"列举目录/章节"的意图
	outlineExplicitHints = []string{"包含哪些章节", "都包含哪些章节", "有哪些章节", "有哪些小节", "包含哪些小节", "章节列表", "小节列表", "子章节", "子小节", "下级章节", "展开目录", "列出目录"}

	// itemHints 表达"第 N 步 / 第 N 项"的具体条目定位需求
	itemHints = []string{"哪一步", "哪一项", "第几步", "第几项", "具体步骤", "步骤中的"}

	// analyticStrongHints 表达分析/对比/解释意图的强暗示
	analyticStrongHints = []string{"为什么", "原因", "可能原因", "影响", "区别", "对比", "比较", "如何理解", "怎么理解", "说明了什么", "是否意味着", "是否说明", "分析", "解释"}

	// analyticWeakRelationHints 表达"关系/联系"的弱暗示，需与结构线索共同判断
	analyticWeakRelationHints = []string{"关系", "关联", "联系", "相关"}

	// structuralRelationHints 结构关系的常见表述
	structuralRelationHints = []string{
		"前后关系", "相邻关系", "上下级关系", "父子关系", "目录关系", "章节关系",
		"所属关系", "位置关系", "顺序关系", "属于哪个章节", "上级章节", "下级章节",
		"同级章节", "父章节", "子章节",
	}
	// graphOnlyBlockingAnalyticHints 一旦命中，必须走分析而非 GRAPH_ONLY 的关键词
	graphOnlyBlockingAnalyticHints = []string{"为什么", "原因", "可能原因", "影响", "区别", "对比", "比较", "如何理解", "怎么理解", "说明了什么", "是否意味着", "是否说明", "分析", "解释"}

	// graphOnlyContentHints 仅在 "内容/流程" 类问题中出现，用于在纯结构问题时降权
	graphOnlyContentHints = []string{"内容", "要求", "规定", "流程", "步骤", "处理", "执行", "怎么做", "讲了什么", "写了什么", "说了什么"}

	// graphOnlyDirectionHints 相邻关系的方向词（前面/后面/上下/前后 ……）
	graphOnlyDirectionHints = []string{"前面", "后面", "上面", "下面", "之前", "之后", "此前", "随后", "后续", "接着", "紧接着", "往前", "往后", "前一个", "后一个", "上一个", "下一个", "上一", "下一", "相邻", "前后", "顺序", "位置", "属于", "上级", "父章节", "同级"}

	// graphOnlyExplicitAdjacencyHints 明确表达"相邻章节"的结构词
	graphOnlyExplicitAdjacencyHints = []string{"前一个", "后一个", "上一个", "下一个", "相邻", "前后", "顺序", "位置", "属于", "上级", "父章节", "同级"}

	// graphOnlyStructureObjectHints 以"章节/小节/部分"等结构对象为核心的提问
	graphOnlyStructureObjectHints = []string{"章节", "小节", "这章", "这节", "这部分", "这一章", "该章", "本章", "标题", "目录", "部分", "模块", "节点", "条目"}

	// graphOnlyOutlineActionHints 表达"展开/包含哪些/列出"等目录动作的词
	graphOnlyOutlineActionHints = []string{"下面", "下级", "子章节", "子小节", "子项", "展开", "包含哪些", "包括哪些", "有哪些", "列出", "列一下", "组成", "目录"}

	// graphOnlyPronounAnchorHints 以代词为锚点的提问（这个/该/它/刚才……）
	graphOnlyPronounAnchorHints = []string{"这个", "该", "它", "刚才", "上述", "上面"}

	// graphOnlyAdjacencyAnswerHints 指向相邻章节答案目标的提问（哪一节/哪一章……）
	graphOnlyAdjacencyAnswerHints = []string{"哪一节", "哪一章", "哪个章节", "哪个小节", "哪个标题", "哪部分", "哪块"}
)

// ============================================================
// 内部意图结构体
// ============================================================

// graphOnlyIntentDecision 是否进入 GRAPH_ONLY 模式的判定结果
type graphOnlyIntentDecision struct {
	matched    bool    // 是否命中结构图直答
	action     string  // 建议动作：SECTION_ADJACENCY_LOOKUP / CHILD_SECTION_DESCEND
	confidence float64 // 置信度
	reason     string  // 人类可读的判断理由
	source     string  // 判定来源：rule-xxx / llm-xxx / none
}

// questionIntentDecision 文档问题的综合意图分类结果
type questionIntentDecision struct {
	graphOnly       *graphOnlyIntentDecision // 是否进入结构图直答
	analytic        bool                     // 是否为分析/对比/解释型问题
	outline         bool                     // 是否为目录/子章节展开问题
	itemLookup      bool                     // 是否为"第 N 项/第 N 步"型条目定位
	structureHint   bool                     // 是否带有结构线索（章节/编号/引用）
	contentQuestion bool                     // 是否包含"内容/流程/怎么做"等关键字
	confidence      float64                  // 综合置信度
	reason          string                   // 判断理由
	source          string                   // 判定来源
}

// DocumentQuestionRouter 在某个文档内部进行意图判断与章节定位，最终输出导航决策
type DocumentQuestionRouter struct {
	chatModel             *logic.ChatModelImpl[*schema.AgenticMessage] // 可选：兜底意图分类用的对话模型
	structureGraphQuerier graph.StructureGraphQuerier                  // 结构图谱查询能力
	navigationIndexSvc    NavigationIndexService                       // 可选：章节索引服务；非 nil 时用于章节定位
	promptTemplateLogic   logic.PromptTemplateLogic                    // 可选：LLM 用的 Prompt 模板渲染
}

// NavigationIndexService 可选的章节索引服务接口（与结构图谱并列定位章节）
type NavigationIndexService interface {
	// SearchSections 按关键词+维度检索匹配的章节命中
	SearchSections(ctx context.Context, documentId int64, topic, facet, informationNeed, question string, topK int) ([]*NavigationSectionHit, error)
}

// NavigationSectionHit 章节索引服务返回的命中节点
type NavigationSectionHit struct {
	NodeId int64   // 结构图谱中的节点 ID
	Score  float64 // 命中分数
}

// NewDocumentQuestionRouter 构造文档问题路由器
func NewDocumentQuestionRouter(
	chatModel *logic.ChatModelImpl[*schema.AgenticMessage],
	structureGraphQuerier graph.StructureGraphQuerier,
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

// Route 在用户问题进行意图用户问题进行意图判断与章节定位，返回导航决策。
//
// 总体流程：
//  1. 规范化输入（改写问题、子问题列表、拼接路由文本）
//  2. 识别意图（本地规则 + 可选 LLM 兜底）
//  3. 按 "GRAPH_ONLY → 编号项定位 → 混合检索" 的优先级输出导航决策
func (r *DocumentQuestionRouter) Route(ctx context.Context, documentId int64, originalQuestion string, rewriteResult *vo.QuestionRewriteResult) (*vo.DocumentNavigationDecision, error) {
	// 选取改写后的问题，无改写则回退原始问题
	rewrittenQuestion := strutil.Trim(originalQuestion)
	if rewriteResult != nil && strutil.IsNotBlank(rewriteResult.RewrittenQuestion) {
		rewrittenQuestion = strutil.Trim(rewriteResult.RewrittenQuestion)
	}

	// 从改写结果中提取子问题列表，为空则以改写问题作为单元素列表
	subQuestions := normalizeSubQuestions(rewriteResult, rewrittenQuestion)

	// 组装检索计划（供下游检索与答案生成复用）
	retrievalPlan := &vo.RetrievalQuestionPlan{
		RetrievalQuestion: rewrittenQuestion,
		SubQuestions:      subQuestions,
	}

	// 拼接路由文本（原始 + 改写），供后续意图识别和章节编号抽取使用
	routeText := strutil.Trim(strutil.Trim(originalQuestion) + " " + rewrittenQuestion)

	// 本地规则优先做意图粗判；必要时由 LLM 做兜底微调
	questionIntent := r.detectQuestionIntent(ctx, routeText, originalQuestion, rewrittenQuestion, subQuestions)

	// 分支 A：命中 GRAPH_ONLY 且无子问题扩散时，直接走结构图直答（无需证据检索）
	graphOnly := questionIntent.graphOnly
	if graphOnly.matched && len(subQuestions) <= 1 {
		section := r.resolveSection(ctx, documentId, originalQuestion, rewrittenQuestion)
		return r.buildDecision(vo.ExecutionModeGraphOnly, graphOnly.action, section, nil, retrievalPlan, graphOnly.reason), nil
	}

	// 分支 B：明确的编号项/步骤型问题（含 "第N步 / 第N项" 或命中 item 关键词）
	// 先通过图定位章节锚点，再交给证据检索，形成 "图 → 证据" 的执行路径
	itemIndex := resolveExplicitItemIndex(routeText)
	itemLookupMatched := itemIndex != nil || questionIntent.itemLookup
	if itemLookupMatched && !questionIntent.analytic {
		section := r.resolveSection(ctx, documentId, originalQuestion, rewrittenQuestion)
		return r.buildDecision(vo.ExecutionModeGraphThenEvidence, vo.DocumentNavigationActionItemReference, section, itemIndex, retrievalPlan, "编号项或步骤型问题走图定位取证"), nil
	}

	// 分支 C：分析型 / 目录型 / 带有结构线索的问题，以及其余普通文档问题
	// 若存在结构线索则尝试定位章节作为软提示（辅助混合检索），否则完全交给向量/关键词混合检索
	var assistedSection *vo2.GraphSection
	needsStructureAssistedRetrieval := questionIntent.analytic || questionIntent.outline || itemIndex != nil || questionIntent.structureHint
	if needsStructureAssistedRetrieval {
		assistedSection = r.resolveSection(ctx, documentId, originalQuestion, rewrittenQuestion)
	}

	// 动作与理由的落地赋值：
	//  - 存在明确的 item 编号时走 ItemReference；否则视为普通 FreshTopic
	//  - 能够给出章节软提示时使用对应理由，否则走通用混合检索理由
	action := vo.DocumentNavigationActionFreshTopic
	if itemIndex != nil {
		action = vo.DocumentNavigationActionItemReference
	}
	reason := "普通文档问题走混合检索"
	if assistedSection != nil {
		reason = "结构线索仅作为软提示辅助混合检索"
	}
	return r.buildDecision(vo.ExecutionModeRetrieval, action, assistedSection, itemIndex, retrievalPlan, reason), nil
}

// ============================================================
// 构建导航决策输出
// ============================================================

// buildDecision 根据执行模式、动作、章节与检索计划，组装最终的 DocumentNavigationDecision
func (r *DocumentQuestionRouter) buildDecision(mode vo.ExecutionMode, action string, section *vo2.GraphSection, itemIndex *int, retrievalPlan *vo.RetrievalQuestionPlan, reason string) *vo.DocumentNavigationDecision {
	decision := &vo.DocumentNavigationDecision{}
	decision.ExecutionMode = mode
	decision.NavigationAction = action
	decision.RetrievalPlan = retrievalPlan

	// 章节存在时，使用 section 作为结构锚点；否则用 "未解析" 占位
	if section != nil {
		decision.StructureAnchor = &vo.ConversationStructureAnchor{
			RootSectionCode:   strutil.Trim(section.NodeCode),
			RootSectionTitle:  strutil.Trim(section.Title),
			TargetSectionHint: strutil.Trim(section.DisplayTitle()),
			StructureNodeId:   section.NodeId,
			CanonicalPath:     section.CanonicalPath,
			ScopeMode:         utils.Ternary(mode == vo.ExecutionModeRetrieval, "SOFT", "GRAPH"),
		}
		decision.SoftSectionHints = []string{section.DisplayTitle()}
	} else {
		decision.StructureAnchor = &vo.ConversationStructureAnchor{
			ScopeMode: utils.Ternary(mode == vo.ExecutionModeRetrieval, "NONE", "GRAPH_UNRESOLVED"),
		}
	}
	// 存在明确的 item 编号锚点（第 N 步 / 第 N 项）时，加入 item 锚点
	if itemIndex != nil {
		decision.ItemAnchor = &vo.ConversationItemAnchor{
			ItemIndex: *itemIndex,
		}
	}

	// 从检索计划 + 章节锚点 + 条目锚点 汇总生成 Query 上下文提示与 Summary 文本
	decision.QueryContextHints = buildQueryHints(retrievalPlan, section, itemIndex)
	decision.SummaryText = buildSummaryText(mode, action, section, itemIndex, reason)

	logx.Infof("文档问答路由完成: mode=%s, action=%s, section=%q, itemIndex=%+v, reason=%q",
		mode.Name(), action, safeDisplayTitle(section), itemIndex, reason)
	return decision
}

// ============================================================
// 问题意图识别（本地规则 + LLM 兜底）
// ============================================================

// detectQuestionIntent 先用本地关键词做粗判，必要时再调用 LLM 做兜底微调。
//
// 执行顺序：
//  1. 空问题快速返回
//  2. 并行计算五类布尔特征（item/analytic/outline/content/structure）
//  3. 满足进入条件时执行 GRAPH_ONLY 本地规则并命中即返回
//  4. 构造本地决策结果，在含糊场景下交由 LLM 兜底
func (r *DocumentQuestionRouter) detectQuestionIntent(ctx context.Context, routeText, originalQuestion, rewrittenQuestion string, subQuestions []string) *questionIntentDecision {
	normalized := strutil.Trim(routeText)
	// 空问题短路，避免后续无意义的规则与 LLM 开销
	if strutil.IsBlank(normalized) {
		return &questionIntentDecision{
			graphOnly: noGraphOnlyIntent("问题为空，跳过路由意图判断。"),
			reason:    "问题为空，跳过路由意图判断。",
			source:    "none",
		}
	}

	// 并行计算五类布尔特征，供后续各分支复用
	//  - itemLookup：是否存在 "第N步 / 第N项" 等条目型引用
	//  - analytic：是否为分析/对比/解释型问题（命中强关键词或弱关系词但非结构关系）
	//  - outline：是否为章节/子章节展开类问题
	//  - contentQuestion：是否提及 "内容/流程/怎么做" 等具体内容关键词
	//  - structureHint：是否包含章节、编号、引用、锚点等结构线索
	itemLookup := looksExplicitItemQuestion(normalized)
	analytic := looksAnalyticQuestion(normalized)
	outline := asksOutline(normalized)
	contentQuestion := strutil.ContainsAny(normalized, graphOnlyContentHints)
	structureHint := mentionsStructure(normalized) || hasGraphOnlyAnchor(normalized) || outline
	// 初始 GRAPH_ONLY 结果为 "未命中"，后续可能被覆盖
	graphOnly := noGraphOnlyIntent("本地规则未命中结构图直答意图。")

	// 判断是否允许尝试 GRAPH_ONLY 本地规则
	// 准入条件：无子问题扩散 ∧ 非条目型 ∧ 非内容型 ∧ 若为分析型则不能命中阻塞型分析关键词
	canTryGraphOnlyRules := len(subQuestions) <= 1 && !itemLookup && !contentQuestion && !(analytic && strutil.ContainsAny(normalized, graphOnlyBlockingAnalyticHints))
	if canTryGraphOnlyRules {
		graphOnly = r.detectGraphOnlyIntentByRules(normalized)
	}

	// 本地规则命中 GRAPH_ONLY 强模式时立即返回（信任度高，无需 LLM）
	// 注：outline 字段在命中 ChildSectionDescend 时被强制置真，以保证下游识别为目录展开意图
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

	// 本地规则未给出明确结论，构造标准本地决策
	localDecision := &questionIntentDecision{
		graphOnly:       graphOnly,
		analytic:        analytic,
		outline:         outline,
		itemLookup:      itemLookup,
		structureHint:   structureHint,
		contentQuestion: contentQuestion,
		// 置信度取经验阈值：本地规则在多数常规问题上可接受但非绝对可信
		confidence: 0.65,
		reason:     "本地路由意图规则判断完成。",
		source:     "local-rules",
	}

	// 若当前问题命中 LLM 兜底触发条件（含糊、关系型、含导航线索），
	// 则调用 LLM 做二次分类；否则直接返回本地决策
	if !shouldUseLLMQuestionIntent(normalized, subQuestions, localDecision) {
		return localDecision
	}

	return r.classifyQuestionIntentWithModel(ctx, originalQuestion, rewrittenQuestion, normalized, localDecision)
}

// detectGraphOnlyIntentByRules 根据本地规则判断是否允许进入 GRAPH_ONLY，并给出动作建议。
//
// 规则按 "置信度从高到低" 的顺序匹配，先命中先返回：
//  1. 相邻章节关系规则（5 个子规则，动作：SectionAdjacencyLookup）
//  2. 目录/子章节展开规则（2 个子规则，动作：ChildSectionDescend）
//  3. 均未命中时返回 "未命中"，交由下游继续处理
func (r *DocumentQuestionRouter) detectGraphOnlyIntentByRules(question string) *graphOnlyIntentDecision {
	// 最明确的相邻章节表达（上一节/下一章/属于哪个章节……）→ 最高置信度直答
	if strutil.ContainsAny(question, adjacencyHints) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionSectionAdjacencyLookup,
			confidence: 1.0,
			reason:     "命中明确相邻章节表达，结构型问题直接走图查询。",
			source:     "rule-adjacency-hint",
		}
	}

	// 预先抽取三个公共布尔特征，供后续多个相邻章节规则共享
	//  - hasSectionReference：问题中是否出现章节编号（1.2 / 第3章）
	//  - hasExplicitAdjacency：是否出现前后关系、同级关系等方向关系词
	//  - hasAdjacencyAnswerTarget：是否以 "哪一节/哪一章" 等作为答案目标
	hasSectionCode := sectionCodePattern.MatchString(question)
	hasChineseReference := chineseSectionReferencePattern.MatchString(question)
	hasSectionReference := hasSectionCode || hasChineseReference
	hasExplicitAdjacency := strutil.ContainsAny(question, graphOnlyExplicitAdjacencyHints)
	hasAdjacencyAnswerTarget := strutil.ContainsAny(question, graphOnlyAdjacencyAnswerHints)

	// 章节编号 + 方向词/答案目标组合
	if hasSectionReference && (hasExplicitAdjacency || hasAdjacencyAnswerTarget) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionSectionAdjacencyLookup,
			confidence: 0.92,
			reason:     "命中章节编号与方向词组合，结构相邻关系问题走图查询。",
			source:     "rule-section-code-direction",
		}
	}
	// 引号锚点（标题引用）+ 方向词/答案目标组合
	if quotedTextPattern.MatchString(question) && (hasExplicitAdjacency || hasAdjacencyAnswerTarget) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionSectionAdjacencyLookup,
			confidence: 0.9,
			reason:     "命中标题锚点与方向词组合，结构相邻关系问题走图查询。",
			source:     "rule-quoted-title-direction",
		}
	}
	// 代词锚点（这个/该/它）+ 方向词 + 章节答案目标，三要素齐全
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
	// 章节/小节等结构对象词 + 方向关系词的宽松组合
	if strutil.ContainsAny(question, graphOnlyStructureObjectHints) && strutil.ContainsAny(question, graphOnlyExplicitAdjacencyHints) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionSectionAdjacencyLookup,
			confidence: 0.86,
			reason:     "命中结构对象与方向关系组合，结构相邻关系问题走图查询。",
			source:     "rule-structure-direction",
		}
	}

	// 最明确目录展开表达（包含哪些章节/列出目录……）→ 最高置信度
	if asksOutline(question) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionChildSectionDescend,
			confidence: 1.0,
			reason:     "命中明确章节展开表达，结构型问题直接走图查询。",
			source:     "rule-outline-hint",
		}
	}
	// 存在章节锚点 + 目录展开动作词（下级/子章节/有哪些……）的组合
	if hasGraphOnlyAnchor(question) && strutil.ContainsAny(question, graphOnlyOutlineActionHints) {
		return &graphOnlyIntentDecision{
			matched:    true,
			action:     vo.DocumentNavigationActionChildSectionDescend,
			confidence: 0.86,
			reason:     "命中章节锚点与目录展开动作，结构型问题直接走图查询。",
			source:     "rule-outline-action",
		}
	}

	// 所有本地规则均未命中，返回占位结果
	return noGraphOnlyIntent("本地规则未命中结构图直答意图。")
}

// shouldUseLLMQuestionIntent 判断是否值得调用 LLM 做兜底。
//
// 设计原则：
//   - 本地规则已经给出明确结论时（多子问题 / 明确条目型 / 内容型 / 强分析型）跳过，避免不必要开销
//   - 仅在问题含糊但带有章节导航线索、或含糊的关系型表达时，才交由 LLM 二次判断
func shouldUseLLMQuestionIntent(question string, subQuestions []string, decision *questionIntentDecision) bool {
	// 排除 1：存在多子问题时由上层循环分别处理，不重复走 LLM
	if len(subQuestions) > 1 {
		return false
	}
	// 排除 2：明确条目型或内容型问题 → 本地规则足够
	if decision.itemLookup || decision.contentQuestion {
		return false
	}
	// 排除 3：强分析型问题（命中阻塞关键词：为什么/原因/影响……）→ 本地规则已足够
	if decision.analytic && strutil.ContainsAny(question, graphOnlyBlockingAnalyticHints) {
		return false
	}

	// 准入 1：问题同时包含章节锚点 + 导航线索（方向词/目录展开词）→ 让 LLM 判断是否为结构图意图
	hasGraphOnlyNavigationCue := strutil.ContainsAny(question, graphOnlyDirectionHints) || strutil.ContainsAny(question, graphOnlyOutlineActionHints)
	if hasGraphOnlyAnchor(question) && hasGraphOnlyNavigationCue {
		return true
	}
	// 准入 2：关系型表达（关系/关联/相关）+ 结构线索 → 交给 LLM 判断是结构关系还是语义关系
	return strutil.ContainsAny(question, analyticWeakRelationHints) && decision.structureHint
}

// ============================================================
// LLM 兜底意图检测
// ============================================================

// llmQuestionIntentPayload 模型输出的结构化 JSON，用于反向解析意图
type llmQuestionIntentPayload struct {
	IntentType    string  `json:"intent_type"` // ADJACENCY / OUTLINE / ANALYTIC / ITEM_LOOKUP / CONTENT_QA …
	Action        string  `json:"action"`      // 可选：具体动作
	GraphOnly     bool    `json:"graph_only"`
	Analytic      bool    `json:"analytic"`
	Outline       bool    `json:"outline"`
	ItemLookup    bool    `json:"item_lookup"`
	ContentQA     bool    `json:"content_qa"`
	StructureHint bool    `json:"structure_hint"`
	Confidence    float64 `json:"confidence"`
	Reason        string  `json:"reason"`
}

// classifyQuestionIntentWithModel 调用 LLM 对含糊问题做兜底分类；chatModel 为空时回退本地决策
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

// parseQuestionIntentResult 从模型返回文本中抽取 JSON 并解析；置信度不足或解析失败时回退本地决策
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
	if confidence < graphOnlyThreshold {
		return localDecision
	}
	reason := utils.BlankToDefault(strutil.Trim(payload.Reason), "LLM 判定完成。")
	action := resolveModelGraphOnlyAction(payload.Action, payload.IntentType)
	acceptGraphOnly := payload.GraphOnly && action != "" &&
		(strings.EqualFold(payload.IntentType, "ADJACENCY") || strings.EqualFold(payload.IntentType, "OUTLINE"))

	graphOnly := noGraphOnlyIntent("LLM 判定不适合结构图直答: " + reason)
	if acceptGraphOnly {
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

// resolveModelGraphOnlyAction 将模型返回的动作/意图映射到我们内部的动作常量
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

// extractJsonObject 从字符串中抽取大括号包裹的 JSON 对象；找不到时回退原文
func extractJsonObject(raw string) string {
	trimmed := strutil.Trim(raw)
	if matched := jsonObjectPattern.FindString(trimmed); matched != "" {
		return matched
	}
	return trimmed
}

// normalizeConfidence 将模型返回的可能超出 0-1 的置信度规范化到 [0, 1)，再由调用方与阈值比较
func normalizeConfidence(confidence float64) float64 {
	if confidence > 1 {
		return confidence / 100.0
	}
	return max(0, confidence)
}

// ============================================================
// 章节定位
// ============================================================

// resolveSection 按 "章节编号 → 索引服务 → 本地短语打分 → 图谱 BestSection" 的顺序定位章节。
// 设计思想：置信度由高到低依次尝试，命中即返回，避免后续更低成本策略。
//  1. 通过显式章节编号（如 1.2 / 第3章）直接定位 → 最高置信度
//  2. 通过可选章节索引服务（navigationIndexSvc）检索 → 依赖外部索引
//  3. 从问题中抽取短语，对文档内章节本地打分匹配 → 纯本地策略
//  4. 回退到图谱服务的 FindBestSection（一般基于向量/关键词检索）→ 最终兜底
func (r *DocumentQuestionRouter) resolveSection(ctx context.Context, documentId int64, originalQuestion, rewrittenQuestion string) *vo2.GraphSection {
	// 步骤 0：入参/依赖校验 — 无 documentId 或无结构图谱查询器时直接放弃
	if documentId == 0 || r.structureGraphQuerier == nil {
		return nil
	}
	// 步骤 1：章节编号直接定位（最高置信度）
	section := r.resolveBySectionCode(ctx, documentId, originalQuestion, rewrittenQuestion)
	if section != nil {
		return section
	}
	// 步骤 2：章节索引服务检索（可选依赖，可能为 nil）
	section = r.resolveByNavigationIndex(ctx, documentId, originalQuestion, rewrittenQuestion)
	if section != nil {
		return section
	}
	// 步骤 3：从问题中抽取短语，对文档内章节做本地打分匹配
	phrases := r.buildSectionPhrases(originalQuestion, rewrittenQuestion)
	section = r.resolveByLocalStructure(ctx, documentId, phrases)
	if section != nil {
		return section
	}
	// 步骤 4：最终兜底 — 调用图谱服务的 FindBestSection（一般由图谱实现做向量/关键词混合检索）
	section, err := r.structureGraphQuerier.FindBestSection(ctx, documentId, rewrittenQuestion, "")
	if err != nil {
		logx.Errorf("FindBestSection 调用失败: documentId=%d, err=%v", documentId, err)
		return nil
	}
	return section
}

// resolveBySectionCode 从问题文本中抽取章节编号（小数编号 / 中文 "第X章"）后通过图谱定位。
// 抽取顺序：先小数编号（如 "1.2"），再中文编号（如 "第3章"）；任一命中即返回。
func (r *DocumentQuestionRouter) resolveBySectionCode(ctx context.Context, documentId int64, originalQuestion, rewrittenQuestion string) *vo2.GraphSection {
	// 合并原始与改写问题，提高命中概率
	combined := strutil.Trim(originalQuestion) + " " + strutil.Trim(rewrittenQuestion)

	// 子步骤 1：按 "1.2.3" 类小数编号正则抽取，逐条在图谱中查找
	for _, code := range sectionCodePattern.FindAllString(combined, -1) {
		section, err := r.structureGraphQuerier.FindSectionByCode(ctx, documentId, code)
		if err == nil && section != nil {
			return section
		}
	}
	// 子步骤 2：按 "第 3 章 / 第三节" 中文编号正则抽取，先解析为阿拉伯数字再在图谱中查找
	for _, m := range chineseSectionReferencePattern.FindAllStringSubmatch(combined, -1) {
		// 正则捕获组长度不足，跳过
		if len(m) < 2 {
			continue
		}
		// 中文数字 → 阿拉伯数字转换失败（如为 0 或非数字），跳过
		parsed := utils.ParseChineseNumber(m[1])
		if parsed <= 0 {
			continue
		}
		code := strconv.Itoa(parsed)
		if section, err := r.structureGraphQuerier.FindSectionByCode(ctx, documentId, code); err == nil && section != nil {
			return section
		}
	}
	// 未抽取到可命中的章节编号，返回 nil 由上层继续尝试其他策略
	return nil
}

// resolveByLocalStructure 用从问题中抽取的短语对文档内章节打分，返回分数最高且 >= 45 的章节。
// 打分策略：每个章节的 Title / SectionPath / AnchorText / ContentText 分别与短语做包含匹配，
// 取所有短语中最高分值作为该章节得分（命中路径/标题优先于命中锚点/正文）。
func (r *DocumentQuestionRouter) resolveByLocalStructure(ctx context.Context, documentId int64, phrases []string) *vo2.GraphSection {
	// 无候选短语 → 无法打分，直接返回
	if len(phrases) == 0 {
		return nil
	}
	// 从图谱获取文档全部章节列表；失败或为空时无法继续
	sections, err := r.structureGraphQuerier.ListSections(ctx, documentId)
	if err != nil || len(sections) == 0 {
		return nil
	}

	// 遍历章节，维护最高得分章节；阈值 45 为经验值（命中正文的最低加分即 45），用于过滤噪声
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

// resolveByNavigationIndex 通过可选的章节索引服务定位节点，命中时将最高分数节点转化为图谱章节。
// 注：索引服务可能未配置（navigationIndexSvc 为 nil），该函数在此场景下直接返回 nil。
func (r *DocumentQuestionRouter) resolveByNavigationIndex(ctx context.Context, documentId int64, originalQuestion, rewrittenQuestion string) *vo2.GraphSection {
	// 依赖校验：索引服务或图谱服务未配置则跳过此策略
	if r.navigationIndexSvc == nil || r.structureGraphQuerier == nil {
		return nil
	}
	// 以改写后的问题为主要查询词，缺失时回退到原始问题
	query := utils.BlankToDefault(rewrittenQuestion, originalQuestion)
	// 调用索引服务：传入查询词 + facet（粗略维度，用于优化索引扫描）+ 取前 5 条候选
	hits, err := r.navigationIndexSvc.SearchSections(ctx, documentId, query, detectFacet(query), "", query, 5)
	if err != nil || len(hits) == 0 {
		return nil
	}
	// 将索引服务返回的首条（最高分）节点 ID 映射为图谱章节
	section, err := r.structureGraphQuerier.FindSectionById(ctx, documentId, hits[0].NodeId)
	return section
}

// ============================================================
// 章节短语抽取与打分
// ============================================================

// buildSectionPhrases 从原始/改写问题中抽取用于章节匹配的短语列表（上限 8）。
// 抽取策略按 "精确 → 暗示" 的顺序进行，短语先去重后裁剪，最后做长度过滤：
//  1. 原始问题与改写问题自身（全量候选）
//  2. 引号包裹的短语（用户显式的标题引用）
//  3. 相邻章节/目录展开标记词之前的片段（潜在章节标题）
//  4. "第N步" 等步骤标记之前的片段（潜在步骤所在章节）
func (r *DocumentQuestionRouter) buildSectionPhrases(originalQuestion, rewrittenQuestion string) []string {
	// seen + addIfAbsent 组合：保证短语去重、清洗空白，且最终上限为 8
	seen := make(map[string]bool)
	phrases := make([]string, 0, 8)
	// 内部辅助函数：清洗短语，若非空且未见过则加入列表（去重 + 清洗一步完成）
	addIfAbsent := func(p string) {
		cleaned := cleanPhrase(p)
		if strutil.IsNotBlank(cleaned) && !seen[cleaned] {
			seen[cleaned] = true
			phrases = append(phrases, cleaned)
		}
	}

	// 候选 1：原始与改写问题自身（最粗略的候选，用于全词匹配）
	addIfAbsent(originalQuestion)
	addIfAbsent(rewrittenQuestion)
	// 候选 2：引号包裹的短语（用户显式引用的标题）
	for _, q := range extractQuotedPhrases(originalQuestion) {
		addIfAbsent(q)
	}
	for _, q := range extractQuotedPhrases(rewrittenQuestion) {
		addIfAbsent(q)
	}

	// 候选 3：相邻章节/目录展开标记词之前的片段（"xxx 的上一节" 中的 xxx）
	for _, marker := range adjacencyHints {
		addIfAbsent(textBeforeMarker(originalQuestion, marker))
		addIfAbsent(textBeforeMarker(rewrittenQuestion, marker))
	}
	for _, marker := range outlineExplicitHints {
		addIfAbsent(textBeforeMarker(originalQuestion, marker))
		addIfAbsent(textBeforeMarker(rewrittenQuestion, marker))
	}
	// 候选 4："第N步" 等步骤标记之前的片段（潜在指代某个流程所在章节）
	combined := strutil.Trim(originalQuestion) + " " + strutil.Trim(rewrittenQuestion)
	for _, step := range stepReferencePattern.FindAllString(combined, -1) {
		addIfAbsent(textBeforeMarker(originalQuestion, step))
		addIfAbsent(textBeforeMarker(rewrittenQuestion, step))
	}

	// 归一化 & 过滤：仅保留归一化后 rune 数 >= 2 的短语；上限 8，避免下游打分膨胀
	filtered := make([]string, 0, len(phrases))
	for _, p := range phrases {
		if utf8.RuneCountInString(normalizeForSection(p)) >= 2 {
			filtered = append(filtered, p)
		}
		if len(filtered) >= 8 {
			break
		}
	}
	return filtered
}

// scoreSection 对单个章节按标题 / 显示路径 / 锚点文本 / 正文叠加打分。
// 评分规则：对每个候选短语在章节的四个字段中做 "包含匹配"，取最高得分作为章节分数。
// 基础分 + 短语长度的线性加权；路径/标题权重最高，正文最低（正文匹配噪声较大）。
func scoreSection(section *vo2.GraphSection, phrases []string) float64 {
	// 空章节或无短语，直接返回 0
	if section == nil || len(phrases) == 0 {
		return 0
	}
	// 预先对章节的四个字段做归一化（小写 + 清除分隔符），避免在循环中重复处理
	title := normalizeForSection(section.Title)
	path := normalizeForSection(section.SectionPath)
	anchor := normalizeForSection(section.AnchorText)
	content := normalizeForSection(section.ContentText)

	// best 取所有短语中最高的命中分数（而非累加，避免多短语的长章节被无限拉高）
	var best float64
	for _, phrase := range phrases {
		normalized := normalizeForSection(phrase)
		normalizedLen := float64(utf8.RuneCountInString(normalized))
		// 过短短语（< 2 rune）跳过，避免误匹配
		if normalizedLen < 2 {
			continue
		}
		// 命中路径/标题 → 高置信度；命中锚点 → 中等置信度；命中正文 → 最低置信度
		// 注意：使用 max 而非累加，以确保 "单一高置信度命中" 优于 "多次弱命中"
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
			// 正文命中封顶 20 个字符的加分，避免长句无意义拉高
			best = max(best, 45.0+min(normalizedLen, 20.0))
		}
	}
	return best
}

// ============================================================
// 小工具函数：意图/短语/文本/摘要
// ============================================================

// looksExplicitItemQuestion 判断问题是否涉及 "第N步 / 第N项" 等条目型引用。
//
// 判定逻辑：
//   - 命中 itemHints（如 "哪一步"、"第几项"）直接返回 true
//   - 或能够从问题中抽取到显式的条目编号（如第 3 步）也返回 true
func looksExplicitItemQuestion(question string) bool {
	// 分支 1：命中条目型关键词（最宽泛的匹配）
	if strutil.ContainsAny(question, itemHints) {
		return true
	}
	// 分支 2：能够抽取到显式的条目编号（严格匹配，需要正则识别）
	return resolveExplicitItemIndex(question) != nil
}

// looksAnalyticQuestion 判断问题是否为分析/对比/解释型。
//
// 判定逻辑：
//   - 命中强关键词（为什么 / 原因 / 区别 / 对比 …）直接为 true
//   - 命中弱关系关键词（关系 / 关联 / 相关）时，排除结构关系后返回 true
//   - 否则为 false
func looksAnalyticQuestion(question string) bool {
	// 强关键词：直接判定为分析型
	if strutil.ContainsAny(question, analyticStrongHints) {
		return true
	}
	// 弱关系关键词：若不出现则直接排除；若出现则进一步确认是否为结构关系（非结构关系才算分析）
	if !strutil.ContainsAny(question, analyticWeakRelationHints) {
		return false
	}
	return !looksStructuralRelationQuestion(question)
}

// looksStructuralRelationQuestion 判断问题是否为 "结构关系"（章节之间的关系，而非内容关系）。
//
// 判定逻辑：
//   - 命中结构关系关键词（前后关系 / 上下级关系 / 章节关系 …）
//   - 或命中章节锚点 + 方向词组合
func looksStructuralRelationQuestion(question string) bool {
	// 分支 1：显式结构关系关键词命中
	if strutil.ContainsAny(question, structuralRelationHints) {
		return true
	}
	// 分支 2：必须先有锚点（章节/编号/代词），再加上方向关系词，才能判定为结构关系
	if !hasGraphOnlyAnchor(question) {
		return false
	}
	return strutil.ContainsAny(question, graphOnlyExplicitAdjacencyHints) || strutil.ContainsAny(question, graphOnlyDirectionHints)
}

// mentionsStructure 判断问题是否涉及结构化内容（章节、小节、条目、步骤等关键词，或显式编号/引号引用）。
// 供意图识别阶段快速判断是否需要尝试结构定位。
func mentionsStructure(question string) bool {
	return strutil.ContainsAny(question, []string{"章节", "小节", "条目", "步骤", "项"}) ||
		quotedTextPattern.MatchString(question) ||
		sectionCodePattern.MatchString(question)
}

// hasGraphOnlyAnchor 判断问题中是否存在可作为 GRAPH_ONLY 锚点的线索。
//
// 锚点包括：
//   - 小数编号（如 1.2）
//   - 中文编号（如 第3章）
//   - 引号引用（如 "xxx"）
//   - 结构对象词（章节 / 小节 / 这章 …）
//   - 代词锚点（这个 / 该 / 它 / 刚才 …）
func hasGraphOnlyAnchor(question string) bool {
	return sectionCodePattern.MatchString(question) ||
		chineseSectionReferencePattern.MatchString(question) ||
		quotedTextPattern.MatchString(question) ||
		strutil.ContainsAny(question, graphOnlyStructureObjectHints) ||
		strutil.ContainsAny(question, graphOnlyPronounAnchorHints)
}

// asksOutline 判断问题是否为 "章节/子章节展开" 类问题（如 "包含哪些章节"、"有哪些小节"）。
// 反例：若问题同时包含 "内容/流程" 等具体内容关键词，则视为内容问题而非目录问题。
func asksOutline(question string) bool {
	if strutil.IsBlank(question) {
		return false
	}
	// 过滤：出现具体内容关键词时，不作为纯目录问题（需要走证据检索而非结构图直答）
	if strutil.ContainsAny(question, graphOnlyContentHints) {
		return false
	}
	// 命中目录展开强关键词，或 章节锚点 + 目录展开动作词组合
	return strutil.ContainsAny(question, outlineExplicitHints) ||
		(hasGraphOnlyAnchor(question) && strutil.ContainsAny(question, graphOnlyOutlineActionHints))
}

// normalizeForSection 统一小写并清理分隔/空白字符，便于大小写不敏感的章节匹配
func normalizeForSection(text string) string {
	if strutil.IsBlank(text) {
		return ""
	}
	cleaned := normalizePattern.ReplaceAllString(text, "")
	return strings.ToLower(cleaned)
}

// cleanPhrase 去掉短语中的噪声词（章节/小节/目录/刚才说的等）以及空白
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

// extractQuotedPhrases 抽取引号包裹的短语（“xxx” / "xxx" / 'xxx'）
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

// textBeforeMarker 返回 marker 之前的片段，用于抽取 "xxx 的上一节" 中的候选章节名
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

// normalizeSubQuestions 提取改写结果中的子问题列表；为空则退回原问题作为单元素列表
func normalizeSubQuestions(rewriteResult *vo.QuestionRewriteResult, fallback string) []string {
	if rewriteResult == nil || len(rewriteResult.SubQuestions) == 0 {
		return []string{strutil.Trim(fallback)}
	}
	out := make([]string, 0, len(rewriteResult.SubQuestions))
	seen := make(map[string]bool)
	for _, q := range rewriteResult.SubQuestions {
		trimmed := strutil.Trim(q)
		if strutil.IsNotBlank(trimmed) && !seen[trimmed] {
			seen[trimmed] = true
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{strutil.Trim(fallback)}
	}
	return out
}

// resolveExplicitItemIndex 从问题中抽取 "第N步 / 第N项" 的 N；未抽到返回 nil
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

// detectFacet 从问题中粗略识别维度（章节位置 / 章节 / 步骤 …），供索引服务使用
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

// buildQueryHints 从检索计划 + 章节锚点 + 条目锚点组装 Query 上下文提示（上限 10 条）
func buildQueryHints(retrievalPlan *vo.RetrievalQuestionPlan, section *vo2.GraphSection, itemIndex *int) []string {
	seen := make(map[string]bool)
	hints := make([]string, 0, 10)
	add := func(h string) {
		if h != "" && !seen[h] && len(hints) < 10 {
			seen[h] = true
			hints = append(hints, h)
		}
	}

	if retrievalPlan != nil {
		if strutil.IsNotBlank(retrievalPlan.RetrievalQuestion) {
			for _, term := range splitRoughTerms(retrievalPlan.RetrievalQuestion) {
				add(term)
			}
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
	return hints
}

// splitRoughTerms 将问题按常见分隔符切分为关键词（上限 6 个）
func splitRoughTerms(question string) []string {
	trimmed := strutil.Trim(question)
	if trimmed == "" {
		return nil
	}

	parts := querySplitPattern.Split(trimmed, -1)
	var result []string
	seen := make(map[string]bool)

	for _, p := range parts {
		if len(result) >= 6 {
			break
		}
		word := strutil.Trim(p)
		if utf8.RuneCountInString(word) > 1 && !seen[word] {
			seen[word] = true
			result = append(result, word)
		}
	}
	return result
}

// buildSummaryText 根据模式、动作、章节、条目、理由生成一行可读性强的摘要
func buildSummaryText(mode vo.ExecutionMode, action string, section *vo2.GraphSection, itemIndex *int, reason string) string {
	sectionTitle := safeDisplayTitle(section)
	itemIndexStr := ""
	if itemIndex != nil {
		itemIndexStr = strconv.Itoa(*itemIndex)
	}
	return "mode=" + mode.Name() + "; action=" + action + "; section=" + sectionTitle + "; itemIndex=" + itemIndexStr + "; reason=" + strutil.Trim(reason)
}

func safeDisplayTitle(section *vo2.GraphSection) string {
	if section == nil {
		return ""
	}
	return section.DisplayTitle()
}

func noGraphOnlyIntent(reason string) *graphOnlyIntentDecision {
	return &graphOnlyIntentDecision{
		reason: reason,
		source: "none",
	}
}
