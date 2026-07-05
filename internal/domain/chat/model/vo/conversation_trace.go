package vo

import (
	"time"

	list "github.com/duke-git/lancet/v2/datastructure/list"
)

type ConversationTrace struct {
	id               int64
	conversationId   string
	exchangeId       int64
	traceId          string
	modelUsageTraces *list.CopyOnWriteList[*ChatModelUsageTrace]
	limitStats       *ChatLimitStats
	stageCode        *ConversationTraceStage
	stageLevel       int
	stageState       ConversationTraceStageState
	parentStageId    int64
	summaryText      string
	executionMode    string
	errorMessage     string
	snapshotJson     string
	durationMs       int64
	ragObservation   any // RAG检索观测数据，使用any避免循环导入
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

type StageHandle struct {
	StageId        int64                   // 阶段ID
	ConversationId string                  // 对话ID
	StartTime      time.Time               // 开始时间
	StageCode      *ConversationTraceStage // 阶段代码
}

func NewConversationTrace(conversationId string, exchangeId int64, traceId string) *ConversationTrace {
	return &ConversationTrace{
		conversationId:   conversationId,
		exchangeId:       exchangeId,
		traceId:          traceId,
		modelUsageTraces: list.NewCopyOnWriteList([]*ChatModelUsageTrace{}),
		limitStats:       &ChatLimitStats{},
	}
}

// AddModelUsageTrace 添加模型调用轨迹
func (t *ConversationTrace) AddModelUsageTrace(trace *ChatModelUsageTrace) {
	if trace == nil {
		return
	}
	t.modelUsageTraces.Add(trace)
}

// SnapshotModelUsageTraces 获取模型调用轨迹的快照
func (t *ConversationTrace) SnapshotModelUsageTraces() []*ChatModelUsageTrace {
	return t.modelUsageTraces.SubList(0, t.modelUsageTraces.Size())
}

// ConversationId 获取对话ID
func (t *ConversationTrace) ConversationId() string {
	return t.conversationId
}

// ExchangeId 获取交换ID
func (t *ConversationTrace) ExchangeId() int64 {
	return t.exchangeId
}

// TraceId 获取追踪ID
func (t *ConversationTrace) TraceId() string {
	return t.traceId
}
