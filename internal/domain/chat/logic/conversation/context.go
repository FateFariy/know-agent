package conversation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	list "github.com/duke-git/lancet/v2/datastructure/list"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

var weekdayMap = map[time.Weekday]string{
	time.Monday:    "星期一",
	time.Tuesday:   "星期二",
	time.Wednesday: "星期三",
	time.Thursday:  "星期四",
	time.Friday:    "星期五",
	time.Saturday:  "星期六",
	time.Sunday:    "星期日",
}

const Zone = "Asia/Shanghai"

type Context struct {
	ConversationId                 string                                       // 对话ID
	ExchangeId                     int64                                        // 交换ID
	Question                       string                                       // 用户问题
	ChatMode                       enum.ChatQueryMode                           // 聊天模式
	TraceId                        string                                       // 追踪ID
	SelectedDocumentId             int64                                        // 选中的文档ID
	SelectedDocumentName           string                                       // 选中的文档名
	SelectedTaskId                 int64                                        // 选中的任务ID
	KnowledgeBaseSelectionSnapshot *vo.KnowledgeBaseSelectionSnapshot           // 知识库选择快照
	CurrentDateText                string                                       // 当前日期文本
	ExecutionPlan                  atomic.Pointer[vo.ConversationExecutionPlan] // 执行计划
	DebugTrace                     atomic.Pointer[vo.ChatDebugTrace]            // 调试追踪
	Trace                          *vo.ConversationTrace                        // 追踪记录
	LeaseKey                       string                                       // 租约锁键
	Sink                           adapter.Sink                                 // 消息发送器
	answerBuffer                   strings.Builder                              // 响应内容缓冲区
	mu                             sync.Mutex                                   // 响应内容缓冲区锁
	ThinkingSteps                  *list.CopyOnWriteList[string]                // 思考步骤列表
	References                     *list.CopyOnWriteList[*vo.SearchReference]   // 引用列表
	UsedTools                      *list.CopyOnWriteList[string]                // 已使用的工具集合
	Recommendations                []string                                     // 推荐追问列表
	cache                          *semanticCacheCtx                            // 语义缓存状态（集中管理）
	StartTime                      time.Time                                    // 开始时间（毫秒精度）
	FirstResponseTimeMs            atomic.Int64                                 // 首次响应耗时（毫秒）
	Finalized                      atomic.Bool                                  // 是否已完成
	CancelFunc                     context.CancelFunc                           // 资源释放
}

// Finalize 完善会话上下文：添加 ExchangeId、TraceId、DebugTrace、Trace、LeaseKey
func (c *Context) Finalize(exchange *entity.ChatExchange) {
	c.ExchangeId = exchange.ID
	c.TraceId = utils.GenerateUUIDWithoutHyphen()
	c.DebugTrace.Store(vo.NewChatDebugTrace(nil))
	c.Trace = vo.NewConversationTrace(c.ConversationId, exchange.ID, c.TraceId)
	c.LeaseKey = chatRunningLeasePrefix + c.ConversationId
	c.ThinkingSteps = list.NewCopyOnWriteList[string](nil)
	c.References = list.NewCopyOnWriteList[*vo.SearchReference](nil)
	c.UsedTools = list.NewCopyOnWriteList[string](nil)
	c.StartTime = time.Now()
	loc, _ := time.LoadLocation(Zone)
	currentDate := time.Now().In(loc)
	c.CurrentDateText = fmt.Sprintf("%s（%s）", currentDate.Format(time.DateOnly), weekdayMap[currentDate.Weekday()])
}

// BuildChatExchange 构建会话交换对象
func (c *Context) BuildChatExchange(turnStatus int, errorMsg string) *entity.ChatExchange {
	return &entity.ChatExchange{
		ID:                         c.ExchangeId,
		ConversationId:             c.ConversationId,
		Question:                   c.Question,
		Answer:                     c.Answer(),
		ThinkingSteps:              c.SnapshotThinkingSteps(),
		References:                 c.UniqueReferences(),
		DebugTrace:                 c.DebugTraceJSON(),
		TurnStatus:                 turnStatus,
		ErrorMessage:               errorMsg,
		FirstResponseTimeMs:        c.FirstResponseTimeMs.Load(),
		TotalResponseTimeMs:        time.Since(c.StartTime).Milliseconds(),
		KnowledgeBaseSelectionMode: c.KnowledgeBaseSelectionSnapshot.SelectionModeName(),
		SelectedKnowledgeBaseIds:   c.KnowledgeBaseSelectionSnapshot.SelectedKnowledgeBaseIds,
		SelectedKnowledgeBaseNames: c.KnowledgeBaseSelectionSnapshot.SelectedKnowledgeBaseNames,
		RetrievalConfigSnapshot:    c.KnowledgeBaseSelectionSnapshot.RagRuntimeConfigSnapshot(),
	}
}

func (c *Context) SetExecutePlan(plan *vo.ConversationExecutionPlan) {
	c.ExecutionPlan.Store(plan)
}

// PublishThinking 发布思考事件
func (c *Context) PublishThinking(content string) error {
	if c == nil || utils.IsBlank(content) {
		return nil
	}
	c.ThinkingSteps.AddAll([]string{content})
	return c.Sink.Thinking(content, c.ConversationId, c.ExchangeId)
}

// PublishStatus 发布状态事件
func (c *Context) PublishStatus(content string) error {
	if c == nil || utils.IsBlank(content) {
		return nil
	}
	return c.Sink.Status(content, c.ConversationId, c.ExchangeId)
}

// PublishReferences 发布引用事件
func (c *Context) PublishReferences(refs []*vo.SearchReference) error {
	if c == nil || len(refs) == 0 {
		return nil
	}
	c.References.AddAll(refs)
	return c.Sink.References(refs, c.ConversationId, c.ExchangeId)
}

// PublishText 发布文本事件
func (c *Context) PublishText(content string) error {
	if c == nil || content == "" {
		return nil
	}
	c.WriteAnswerBuffer(content)
	c.FirstResponseTimeMs.CompareAndSwap(0, time.Since(c.StartTime).Milliseconds())
	return c.Sink.Text(content, c.ConversationId, c.ExchangeId)
}

// PublishError 发布错误事件
func (c *Context) PublishError(content string) error {
	if c == nil || content == "" {
		return nil
	}
	return c.Sink.Error(content, c.ConversationId, c.ExchangeId)
}

// PublishRecommendations 发布推荐事件
func (c *Context) PublishRecommendations(recommendations []string) error {
	if c == nil || len(recommendations) == 0 {
		return nil
	}
	return c.Sink.Recommendations(recommendations, c.ConversationId, c.ExchangeId)
}

// PublishFinish 发布完成事件
func (c *Context) PublishFinish() error {
	return c.Sink.Finish(c.ConversationId, c.ExchangeId)
}

// ReleaseResources 释放资源
func (c *Context) ReleaseResources() {
	cancelFunc := c.CancelFunc
	if cancelFunc != nil {
		cancelFunc()
		c.CancelFunc = nil
	}
}

// AddUsedTools 添加已使用的工具
func (c *Context) AddUsedTools(tools ...string) {
	for _, tool := range tools {
		if !c.UsedTools.Contain(tool) && utils.IsNotBlank(tool) {
			c.UsedTools.Add(tool)
		}
	}
}

// SnapshotUsedTools 获取已使用的工具列表的快照
func (c *Context) SnapshotUsedTools() []string {
	return c.UsedTools.SubList(0, c.UsedTools.Size())
}

// UniqueReferences 获取唯一引用列表
func (c *Context) UniqueReferences() []*vo.SearchReference {
	size := c.References.Size()
	if size == 0 {
		return nil
	}
	references := c.References.SubList(0, size)
	return utils.Distinct(references, func(ref *vo.SearchReference) string {
		return ref.UniqueKey()
	})
}

// SnapshotThinkingSteps 获取思考步骤列表的快照
func (c *Context) SnapshotThinkingSteps() []string {
	size := c.ThinkingSteps.Size()
	if size == 0 {
		return nil
	}
	return c.ThinkingSteps.SubList(0, size)
}

// WriteAnswerBuffer 写入响应内容缓冲区
func (c *Context) WriteAnswerBuffer(content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.answerBuffer.WriteString(content)
}

// Answer 获取响应内容缓冲区内容
func (c *Context) Answer() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.answerBuffer.String()
}

// AnswerLength 获取响应内容缓冲区长度（字符数）
func (c *Context) AnswerLength() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return utils.Len(c.answerBuffer.String())
}

// ClarificationOptions 获取澄清选项
func (c *Context) ClarificationOptions() []string {
	if execPlan := c.ExecutionPlan.Load(); execPlan != nil {
		return execPlan.ClarificationOptions
	}
	return nil
}

// DebugTraceJSON 序列化调试轨迹
func (c *Context) DebugTraceJSON() string {
	dt := c.DebugTrace.Load()
	if dt == nil {
		return ""
	}
	return dt.Serialize()
}

// ============================================================================
// 语义缓存状态（集中管理，所有 Stage 单一来源）
// ============================================================================

// semanticCacheCtx 集中保存一次请求中的语义缓存相关状态，避免各 Stage 重复查询与状态不一致
type semanticCacheCtx struct {
	hit        bool
	entry      *entity.ChatCacheEntry
	strategy   enum.ReuseStrategy
	similarity float32
}

// MarkCacheHit 记录命中：挂载缓存条目与相似度
func (c *semanticCacheCtx) MarkCacheHit(h *CacheHit) {
	if c == nil {
		return
	}
	c.hit = true
	c.entry = h.Entry
	c.similarity = h.Similarity
}

// MarkCacheMiss 记录未命中
func (c *semanticCacheCtx) MarkCacheMiss() {
	if c == nil {
		return
	}
	c.hit = false
}

// IsCacheHit 是否已命中语义缓存
func (c *semanticCacheCtx) IsCacheHit() bool {
	return c != nil && c.hit
}

// CacheEntry 返回命中的缓存条目
func (c *semanticCacheCtx) CacheEntry() *entity.ChatCacheEntry {
	if c == nil {
		return nil
	}
	return c.entry
}

// ReuseStrategy 本次请求生效的复用策略
func (c *semanticCacheCtx) ReuseStrategy() enum.ReuseStrategy {
	if c == nil {
		return enum.ReuseRetrievalOnly
	}
	return c.strategy
}

// CacheSimilarity 命中相似度（埋点用）
func (c *semanticCacheCtx) CacheSimilarity() float32 {
	if c == nil {
		return 0
	}
	return c.similarity
}

// applyCachedExecution 命中后将缓存的必要字段回填进当前执行计划，
// 保留当前请求的私有上下文（历史/时间/追问锚点）不被覆盖
func (c *Context) applyCachedExecution(ce *vo.CachedExecution) {
	ep := c.ExecutionPlan.Load()
	if ce == nil || ep == nil {
		return
	}
	ep.RetrievalPlan = ce.RetrievalPlan
	ep.RetrievalResult = ce.RetrievalResult
	ep.PromptAssemblyResult = ce.PromptAssemblyResult
}
