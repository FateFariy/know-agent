package vo

import list "github.com/duke-git/lancet/v2/datastructure/list"

// ChatDebugTrace 单轮对话调试轨迹
type ChatDebugTrace struct {
	ExecutionMode                   string                                `json:"executionMode"`                   // 执行模式
	ChatMode                        ChatQueryMode                         `json:"chatMode"`                        // 聊天模式
	OriginalQuestion                string                                `json:"originalQuestion"`                // 原始问题
	RewriteQuestion                 string                                `json:"rewriteQuestion"`                 // 重写问题
	RewriteSubQuestions             []string                              `json:"rewriteSubQuestions"`             // 重写子问题列表
	RetrievalQuestion               string                                `json:"rewrittenQuestion"`               // 检索问题
	AgentQuestion                   string                                `json:"agentQuestion"`                   // Agent问题
	NavigationDecision              *DocumentNavigationDecision           `json:"navigationDecision"`              // 文档导航决策
	HistorySummary                  string                                `json:"historySummary"`                  // 历史摘要
	LongTermSummary                 string                                `json:"longTermSummary"`                 // 长期摘要
	RecentHistoryTranscript         string                                `json:"recentHistoryTranscript"`         // 近期历史转录
	RecentQuestionTranscript        string                                `json:"RecentQuestionTranscript"`        // 回答近期转录
	QuestionHistoryContext          string                                `json:"questionHistoryContext"`          // 回答历史上下文
	QuestionHistoryFollowUpQuestion bool                                  `json:"questionHistoryFollowUpQuestion"` // 回答历史追问
	HistoryCompressionApplied       bool                                  `json:"historyCompressionApplied"`       // 历史压缩已应用
	HistoryCoveredExchangeId        int64                                 `json:"historyCoveredExchangeId"`        // 历史覆盖交换ID
	HistoryCoveredExchangeCount     int                                   `json:"historyCoveredExchangeCount"`     // 历史覆盖交换数量
	HistoryCompressionCount         int                                   `json:"historyCompressionCount"`         // 历史压缩次数
	CurrentDateText                 string                                `json:"currentDateText"`                 // 当前日期文本
	RequiresRealTimeSearch          bool                                  `json:"requiresRealTimeSearch"`          // 需要实时搜索
	RequiresCurrentDateAnchoring    bool                                  `json:"requiresCurrentDateAnchoring"`    // 需要当前日期锚定
	RetrievalSubQuestions           []string                              `json:"subQuestions"`                    // 检索子问题列表（别名：subQuestions）
	SelectedDocumentId              int64                                 `json:"selectedDocumentId"`              // 选中的文档ID
	SelectedTaskId                  int64                                 `json:"selectedTaskId"`                  // 选中的任务ID
	RetrievalNotes                  *list.CopyOnWriteList[string]         `json:"retrievalNotes"`                  // 检索备注列表
	UsedChannels                    *list.CopyOnWriteList[string]         `json:"usedChannels"`                    // 使用的渠道列表
	ToolTraces                      *list.CopyOnWriteList[*ChatToolTrace] `json:"toolTraces"`                      // 工具调用轨迹列表
	ModelUsageTraces                []*ChatModelUsageTrace                `json:"modelUsageTraces"`                // 模型使用轨迹列表
	LimitStats                      *ChatLimitStats                       `json:"limitStats"`                      // 限制统计
	RagSystemPrompt                 string                                `json:"ragSystemPrompt"`                 // RAG系统提示词
	RagUserPrompt                   string                                `json:"ragUserPrompt"`                   // RAG用户提示词
	NoEvidenceReply                 string                                `json:"noEvidenceReply"`                 // 无证据回复
}

// NewChatDebugTrace 创建新的调试轨迹实例
func NewChatDebugTrace(execPlan *ConversationExecutionPlan) *ChatDebugTrace {
	trace := &ChatDebugTrace{
		RetrievalNotes: list.NewCopyOnWriteList[string](nil),
		UsedChannels:   list.NewCopyOnWriteList[string](nil),
		ToolTraces:     list.NewCopyOnWriteList[*ChatToolTrace](nil),
	}
	if execPlan == nil {
		return trace
	}

	// 基础模式
	if execPlan.Mode != nil {
		trace.ExecutionMode = execPlan.Mode.Name()
	}
	trace.ChatMode = execPlan.ChatMode

	// 问题相关
	trace.OriginalQuestion = execPlan.OriginalQuestion
	trace.RewriteQuestion = execPlan.RewriteQuestion
	trace.RewriteSubQuestions = append(trace.RewriteSubQuestions, execPlan.RewriteSubQuestions...)
	trace.RetrievalQuestion = execPlan.RetrievalQuestion
	trace.AgentQuestion = execPlan.AgentQuestion
	trace.NavigationDecision = execPlan.NavigationDecision

	// 历史摘要
	trace.HistorySummary = execPlan.HistorySummary
	trace.LongTermSummary = execPlan.LongTermSummary
	trace.RecentHistoryTranscript = execPlan.RecentHistoryTranscript
	trace.RecentQuestionTranscript = execPlan.RecentQuestionTranscript
	if execPlan.QuestionHistoryContext != nil {
		trace.QuestionHistoryContext = execPlan.QuestionHistoryContext.RenderedText
		trace.QuestionHistoryFollowUpQuestion = execPlan.QuestionHistoryContext.FollowUpQuestion
	}
	trace.HistoryCompressionApplied = execPlan.HistoryCompressionApplied
	trace.HistoryCoveredExchangeId = execPlan.HistoryCoveredExchangeId
	trace.HistoryCoveredExchangeCount = execPlan.HistoryCoveredExchangeCount
	trace.HistoryCompressionCount = execPlan.HistoryCompressionCount
	trace.CurrentDateText = execPlan.CurrentDateText
	trace.RequiresRealTimeSearch = execPlan.RequiresRealTimeSearch
	trace.RequiresCurrentDateAnchoring = execPlan.RequiresCurrentDateAnchoring

	// 检索子问题
	trace.RetrievalSubQuestions = append(trace.RetrievalSubQuestions, execPlan.RetrievalSubQuestions...)
	trace.SelectedDocumentId = execPlan.SelectedDocumentId
	trace.SelectedTaskId = execPlan.SelectedTaskId

	trace.NoEvidenceReply = execPlan.NoEvidenceReply

	return trace
}

// AddToolTrace 添加工具调用轨迹
func (t *ChatDebugTrace) AddToolTrace(trace *ChatToolTrace) {
	t.ToolTraces.Add(trace)
}

// AddModelUsageTrace 添加模型使用轨迹
func (t *ChatDebugTrace) AddModelUsageTrace(trace *ChatModelUsageTrace) {
	t.ModelUsageTraces = append(t.ModelUsageTraces, trace)
}

// AddUsedChannel 添加使用的渠道
func (t *ChatDebugTrace) AddUsedChannel(channel string) {
	t.UsedChannels.Add(channel)
}

// ChatLimitStats 单轮对话的调用限制统计
type ChatLimitStats struct {
	ModelCallsUsed        int    `json:"modelCallsUsed"`        // 已使用的模型调用次数
	ModelCallsRunLimit    int    `json:"modelCallsRunLimit"`    // 运行限制的模型调用次数
	ModelCallsThreadLimit int    `json:"modelCallsThreadLimit"` // 线程限制的模型调用次数
	ToolCallsUsed         int    `json:"toolCallsUsed"`         // 已使用的工具调用次数
	ToolCallsRunLimit     int    `json:"toolCallsRunLimit"`     // 运行限制的工具调用次数
	ToolCallsThreadLimit  int    `json:"toolCallsThreadLimit"`  // 线程限制的工具调用次数
	LimitTriggered        bool   `json:"limitTriggered"`        // 是否触发限制
	LimitReason           string `json:"limitReason"`           // 限制原因
}

// ChatModelUsageTrace 单次模型调用的使用量轨迹
type ChatModelUsageTrace struct {
	StageName        string  `json:"stageName"`        // 阶段名称
	Provider         string  `json:"provider"`         // 提供商
	Model            string  `json:"model"`            // 模型名称
	PromptTokens     int     `json:"promptTokens"`     // 提示词token数
	CompletionTokens int     `json:"completionTokens"` // 完成token数
	TotalTokens      int     `json:"totalTokens"`      // 总token数
	EstimatedCost    float64 `json:"estimatedCost"`    // 预估成本
	DurationMs       int64   `json:"durationMs"`       // 持续时间毫秒
	Status           string  `json:"status"`           // 状态
}

// ChatToolTrace 单次工具调用观测快照
type ChatToolTrace struct {
	ToolName       string `json:"toolName"`       // 工具名称
	Status         string `json:"status"`         // 状态
	InputSummary   string `json:"inputSummary"`   // 输入摘要
	EffectiveInput string `json:"effectiveInput"` // 有效输入
	OutputSummary  string `json:"outputSummary"`  // 输出摘要
	ErrorMessage   string `json:"errorMessage"`   // 错误信息
	ReferenceCount int    `json:"referenceCount"` // 引用数量
	Topic          string `json:"topic"`          // 主题
	DurationMs     int64  `json:"durationMs"`     // 持续时间毫秒
}

// DocumentNavigationDecision 文档问答路由结果
type DocumentNavigationDecision struct {
	NavigationAction  string                       `json:"navigationAction"`  // 导航动作
	ExecutionMode     ExecutionMode                `json:"executionMode"`     // 执行模式
	StructureAnchor   *ConversationStructureAnchor `json:"structureAnchor"`   // 结构锚点
	ItemAnchor        *ConversationItemAnchor      `json:"itemAnchor"`        // 项目锚点
	RetrievalPlan     *RetrievalQuestionPlan       `json:"retrievalPlan"`     // 检索问题计划
	SummaryText       string                       `json:"summaryText"`       // 摘要文本
	QueryContextHints []string                     `json:"queryContextHints"` // 查询上下文提示
	SoftSectionHints  []string                     `json:"softSectionHints"`  // 软章节提示
}

// ConversationStructureAnchor 会话结构锚点
type ConversationStructureAnchor struct {
	RootSectionCode   string `json:"rootSectionCode"`   // 根章节代码
	RootSectionTitle  string `json:"rootSectionTitle"`  // 根章节标题
	TargetSectionHint string `json:"targetSectionHint"` // 目标章节提示
	StructureNodeId   int64  `json:"structureNodeId"`   // 结构节点ID
	CanonicalPath     string `json:"canonicalPath"`     // 正规路径
	ScopeMode         string `json:"scopeMode"`         // 作用域模式
}

// ConversationItemAnchor 会话项目锚点
type ConversationItemAnchor struct {
	ItemIndex       int    `json:"itemIndex"`       // 项目索引
	ItemText        string `json:"itemText"`        // 项目文本
	StructureNodeId int64  `json:"structureNodeId"` // 结构节点ID
	CanonicalPath   string `json:"canonicalPath"`   // 正规路径
}

// RetrievalQuestionPlan 检索问题计划
type RetrievalQuestionPlan struct {
	MainQuestion      string   `json:"mainQuestion"`      // 主问题
	RetrievalQuestion string   `json:"retrievalQuestion"` // 检索问题
	SubQuestions      []string `json:"subQuestions"`      // 子问题列表
	RetrievalMode     string   `json:"retrievalMode"`     // 检索模式
	MaxResults        int      `json:"maxResults"`        // 最大结果数
	ScoreThreshold    float64  `json:"scoreThreshold"`    // 分数阈值
	ExpandToParent    bool     `json:"expandToParent"`    // 是否扩展到父级
	ExpandToChildren  bool     `json:"expandToChildren"`  // 是否扩展到子级
}

// NewDocumentNavigationDecision 创建新的文档导航决策实例
func NewDocumentNavigationDecision() *DocumentNavigationDecision {
	return &DocumentNavigationDecision{
		QueryContextHints: []string{},
		SoftSectionHints:  []string{},
	}
}
