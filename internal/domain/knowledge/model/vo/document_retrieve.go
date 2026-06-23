package vo

import (
	"regexp"
	"slices"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

var (
	yearPattern    = regexp.MustCompile(`\b(20\d{2})\b`)
	sectionPattern = regexp.MustCompile(`(第\s*[一二三四五六七八九十百0-9]+\s*[章节条部分])|(附录\s*[A-Za-z一二三四五六七八九十0-9]+)`)

	DocumentNameHints = []string{
		"部署手册", "配置手册", "操作手册", "用户手册", "快速开始", "接入指南", "FAQ", "常见问题",
		"说明文档", "说明书", "规范", "指南", "手册", "文档",
	}
	BusinessCategoryHints = []string{
		"流程", "规则", "操作手册", "部署", "配置", "接入", "协议", "故障", "排错", "规范", "说明",
	}
	DocumentTagHints = []string{
		"2024", "2025", "2026", "部署", "配置", "接入", "协议", "FAQ", "故障", "排错", "升级", "兼容",
	}
)

// DocumentRetrieve 文档检索
type DocumentRetrieve struct {
	Question          string                   `json:"question"`          // 问题
	RetrievalQuery    string                   `json:"retrievalQuery"`    // 检索查询
	DocumentId        int64                    `json:"documentId"`        // 文档ID
	TaskId            int64                    `json:"taskId"`            // 任务ID
	DocumentIds       []int64                  `json:"documentIds"`       // 文档ID列表
	TaskIds           []int64                  `json:"taskIds"`           // 任务ID列表
	TopK              int                      `json:"topK"`              // 返回数量
	Filters           *DocumentRetrieveFilters `json:"filters"`           // 过滤器
	QueryContextHints []string                 `json:"queryContextHints"` // 查询上下文提示
}

// DocumentRetrieveFilters 文档检索过滤器
type DocumentRetrieveFilters struct {
	DocumentNameHints     []string `json:"documentNameHints"`     // 文档名称提示
	BusinessCategoryHints []string `json:"businessCategoryHints"` // 业务类别提示
	DocumentTagHints      []string `json:"documentTagHints"`      // 文档标签提示
	SectionPathHints      []string `json:"sectionPathHints"`      // 部分路径提示
	CanonicalPathHints    []string `json:"canonicalPathHints"`    // 正规路径提示
	StructureNodeIdHints  []int64  `json:"structureNodeIdHints"`  // 结构节点ID提示
	ItemIndexHints        []int    `json:"itemIndexHints"`        // 项目索引提示
	YearHints             []string `json:"yearHints"`             // 年份提示
}

func (d *DocumentRetrieveFilters) IsEmpty() bool {
	return len(d.DocumentNameHints) == 0 &&
		len(d.BusinessCategoryHints) == 0 &&
		len(d.DocumentTagHints) == 0 &&
		len(d.SectionPathHints) == 0 &&
		len(d.CanonicalPathHints) == 0 &&
		len(d.StructureNodeIdHints) == 0 &&
		len(d.ItemIndexHints) == 0 &&
		len(d.YearHints) == 0
}

// NewDocumentRetrieve 创建文档检索，根据子问题和执行计划构造检索参数，包含查询增强、过滤器和文档/任务范围
func NewDocumentRetrieve(subQuestion string, plan *vo.ConversationExecutionPlan, topK int) *DocumentRetrieve {
	normalizedQuestion := strutil.Trim(subQuestion)

	// 构建查询增强（将导航提示、上下文提示与原问题合并）
	retrievalQuery, queryContextHints := buildQueryAugmentation(normalizedQuestion, plan)

	// 构建检索过滤器（从问题中提取年份、章节、文档名称等提示）
	filters := buildFilters(normalizedQuestion)

	// 组装检索请求对象
	documentRetrieve := &DocumentRetrieve{
		Question:          normalizedQuestion,
		RetrievalQuery:    retrievalQuery,
		DocumentId:        plan.SelectedDocumentId,
		TaskId:            plan.SelectedTaskId,
		DocumentIds:       plan.RetrievalDocumentIds,
		TaskIds:           plan.RetrievalTaskIds,
		Filters:           filters,
		TopK:              topK,
		QueryContextHints: queryContextHints,
	}

	// 兜底处理 - 当没有检索文档列表时，使用选中的文档ID
	if len(plan.RetrievalDocumentIds) == 0 && plan.SelectedDocumentId != 0 {
		documentRetrieve.DocumentIds = []int64{plan.SelectedDocumentId}
	}

	// 兜底处理 - 当没有检索任务列表时，使用选中的任务ID
	if len(plan.RetrievalTaskIds) == 0 && plan.SelectedTaskId != 0 {
		documentRetrieve.TaskIds = []int64{plan.SelectedTaskId}
	}

	logx.Infof("检索请求构造: originalSubQuestion='%s', retrievalQuery='%s', documentIdList=%v, topK=%d",
		documentRetrieve.Question, documentRetrieve.RetrievalQuery, documentRetrieve.DocumentIds, documentRetrieve.TopK)

	return documentRetrieve
}

func (d *DocumentRetrieve) ResolvedDocumentIDs() []int64 {
	if len(d.DocumentIds) > 0 {
		return d.DocumentIds
	}
	if d.DocumentId != 0 {
		return []int64{d.DocumentId}
	}
	return []int64{}
}

func (d *DocumentRetrieve) ResolvedTaskIDs() []int64 {
	if len(d.TaskIds) > 0 {
		return d.TaskIds
	}
	if d.TaskId != 0 {
		return []int64{d.TaskId}
	}
	return []int64{}
}

// buildQueryAugmentation 构建查询增强，将导航决策提示、历史规划上下文提示与原问题合并，生成更完整的检索查询
func buildQueryAugmentation(normalizedQuestion string, plan *vo.ConversationExecutionPlan) (string, []string) {
	if normalizedQuestion == "" || plan == nil {
		return "", nil
	}

	// 获取导航决策中的查询上下文提示（限制4个）
	var navigationHints []string
	if plan.NavigationDecision != nil {
		navigationHints = distinctTrimLimit(plan.NavigationDecision.QueryContextHints, 4)
	}

	// 从问题中提取有意义的关键词（用于查询提示）
	meaningfulTerms := extractMeaningfulTerms(normalizedQuestion)

	// 判断是否需要历史上下文增强
	// 只有当问题是简短追问且存在历史规划上下文时，才添加历史上下文提示
	if !looksLikeShortFollowUp(normalizedQuestion) || plan.HistoryPlanningContext == nil || len(plan.HistoryPlanningContext.QueryContextHints) == 0 {
		// 非追问场景：仅使用导航提示（如果有）
		if len(navigationHints) == 0 {
			return normalizedQuestion, meaningfulTerms
		}
		retrievalQuery := strutil.Trim(normalizedQuestion + " " + strings.Join(navigationHints, " "))
		queryHints := slices.Concat(navigationHints, meaningfulTerms)
		queryHints = distinctTrimLimit(queryHints, 8)

		return retrievalQuery, queryHints
	}

	// 追问场景：合并历史上下文提示和导航提示
	queryContextHints := distinctTrimLimit(plan.HistoryPlanningContext.QueryContextHints, 4)
	allHints := slices.Concat(queryContextHints, navigationHints)
	allHints = distinctTrimLimit(allHints, 8)

	if len(allHints) == 0 {
		return normalizedQuestion, meaningfulTerms
	}

	// 合并所有提示生成检索查询
	retrievalQuery := strutil.Trim(normalizedQuestion + " " + strings.Join(allHints, " "))
	queryHints := slices.Concat(allHints, meaningfulTerms)
	queryHints = distinctTrimLimit(queryHints, 8)

	return retrievalQuery, queryHints
}

// buildFilters 构建检索过滤器
// 从问题中提取各种提示信息，用于缩小检索范围（文档名称、业务类别、标签、章节路径、年份）
func buildFilters(question string) *DocumentRetrieveFilters {
	if strutil.IsBlank(question) {
		return &DocumentRetrieveFilters{}
	}

	// 标准化问题为小写，用于匹配判断
	normalized := strings.ToLower(question)

	// 提取年份提示（如"2024"、"2025"）
	yearHints := yearPattern.FindAllString(question, -1)

	// 提取章节路径提示（如"第一章"、"附录A"），并去除空格
	sectionPathHints := sectionPattern.FindAllString(question, -1)
	spacePattern := regexp.MustCompile(`\s+`)
	sectionPathHints = stream.FromSlice(sectionPathHints).
		Map(func(item string) string { return spacePattern.ReplaceAllString(item, "") }).
		Filter(func(item string) bool { return item != "" }).ToSlice()

	// 定义匹配谓词（检查问题是否包含提示词）
	predicate := func(index int, item string) bool { return strings.Contains(normalized, strings.ToLower(item)) }

	// 从预定义提示列表中筛选匹配项
	documentNameHints := slice.Filter(DocumentNameHints, predicate)
	businessCategoryHints := slice.Filter(BusinessCategoryHints, predicate)
	documentTagHints := slice.Filter(DocumentTagHints, predicate)

	// 组装过滤器并去重限制数量
	return &DocumentRetrieveFilters{
		DocumentNameHints:     distinctTrimLimit(documentNameHints, 10),
		BusinessCategoryHints: distinctTrimLimit(businessCategoryHints, 10),
		DocumentTagHints:      distinctTrimLimit(documentTagHints, 10),
		SectionPathHints:      distinctTrimLimit(sectionPathHints, 10),
		YearHints:             distinctTrimLimit(yearHints, 10),
	}
}

// looksLikeShortFollowUp 判断是否为简短跟进问题
func looksLikeShortFollowUp(question string) bool {
	keywords := []string{"它", "这个", "那个", "刚才", "前面", "上面"}
	return len([]rune(question)) < 12 || slice.Contain(keywords, question)
}

// extractMeaningfulTerms 提取有意义的词
func extractMeaningfulTerms(question string) []string {
	if strutil.IsBlank(question) {
		return nil
	}
	separators := regexp.MustCompile(`[\s、，,；:：:（）()\-的和及与或]+`)
	segments := separators.Split(question, -1)
	return stream.FromSlice(segments).
		Map(func(s string) string { return strutil.Trim(s) }).
		Filter(func(s string) bool { return len(s) > 1 }).
		Distinct().Limit(6).ToSlice()
}

// distinctTrimLimit 去重并限制数量
func distinctTrimLimit(items []string, limit int) []string {
	if len(items) == 0 {
		return nil
	}
	return stream.FromSlice(items).
		Map(func(s string) string { return strutil.Trim(s) }).
		Filter(func(s string) bool { return s != "" }).
		Distinct().Limit(limit).ToSlice()
}
