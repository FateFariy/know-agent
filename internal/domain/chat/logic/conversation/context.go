package conversation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	list "github.com/duke-git/lancet/v2/datastructure/list"
	"github.com/duke-git/lancet/v2/strutil"

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
	ConversationId       string                                       // 对话ID
	ExchangeId           int64                                        // 交换ID
	Question             string                                       // 用户问题
	ChatMode             enum.ChatQueryMode                           // 聊天模式
	TraceId              string                                       // 追踪ID
	SelectedDocumentId   int64                                        // 选中的文档ID
	SelectedDocumentName string                                       // 选中的文档名
	SelectedTaskId       int64                                        // 选中的任务ID
	CurrentDate          time.Time                                    // 当前日期
	CurrentDateText      string                                       // 当前日期文本
	ExecutionPlan        atomic.Pointer[vo.ConversationExecutionPlan] // 执行计划
	DebugTrace           atomic.Pointer[vo.ChatDebugTrace]            // 调试追踪
	Trace                *vo.ConversationTrace                        // 追踪记录
	LeaseKey             string                                       // 租约锁键
	Sink                 adapter.Sink                                 // 消息发送器
	answerBuffer         strings.Builder                              // 响应内容缓冲区
	mu                   sync.Mutex                                   // 响应内容缓冲区锁
	ThinkingSteps        *list.CopyOnWriteList[string]                // 思考步骤列表
	References           *list.CopyOnWriteList[*vo.SearchReference]   // 引用列表
	UsedTools            *list.CopyOnWriteList[string]                // 已使用的工具集合
	StartTime            time.Time                                    // 开始时间（毫秒精度）
	FirstResponseTimeMs  atomic.Int64                                 // 首次响应耗时（毫秒）
	Finalized            atomic.Bool                                  // 是否已完成
	CancelFunc           context.CancelFunc                           // 资源释放
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
	c.CurrentDate = time.Now().In(loc)
	c.CurrentDateText = fmt.Sprintf("%s（%s）", c.CurrentDate.Format(time.DateOnly), weekdayMap[c.CurrentDate.Weekday()])
}

func (c *Context) SetExecutePlan(plan *vo.ConversationExecutionPlan) {
	c.ExecutionPlan.Store(plan)
}

// PublishThinking 发布思考事件
func (c *Context) PublishThinking(content string) error {
	if c == nil || strutil.IsBlank(content) {
		return nil
	}
	c.ThinkingSteps.AddAll([]string{content})
	return c.Sink.Thinking(content, c.ConversationId, c.ExchangeId)
}

// PublishStatus 发布状态事件
func (c *Context) PublishStatus(content string) error {
	if c == nil || strutil.IsBlank(content) {
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
		if !c.UsedTools.Contain(tool) && strutil.IsNotBlank(tool) {
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

// ExecutionModeName 获取执行模式名称
func (c *Context) ExecutionModeName() string {
	if execPlan := c.ExecutionPlan.Load(); execPlan != nil {
		return execPlan.ExecutionModeName()
	}
	return ""
}

// NeedClarification 是否需要澄清
func (c *Context) NeedClarification() bool {
	if execPlan := c.ExecutionPlan.Load(); execPlan != nil {
		return execPlan.Mode == enum.ExecutionModeClarification && len(execPlan.ClarificationOptions) > 0
	}
	return false
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
