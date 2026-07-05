package vo

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	list "github.com/duke-git/lancet/v2/datastructure/list"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

type ConversationContext struct {
	ConversationId       string                                    // 对话ID
	ExchangeId           int64                                     // 交换ID
	Question             string                                    // 用户问题
	ChatMode             ChatQueryMode                             // 聊天模式
	TraceId              string                                    // 追踪ID
	SelectedDocumentId   int64                                     // 选中的文档ID
	SelectedDocumentName string                                    // 选中的文档名
	SelectedTaskId       int64                                     // 选中的任务ID
	CurrentDate          time.Time                                 // 当前日期
	CurrentDateText      string                                    // 当前日期文本
	ExecutionPlan        atomic.Pointer[ConversationExecutionPlan] // 执行计划
	DebugTrace           atomic.Pointer[ChatDebugTrace]            // 调试追踪
	// todo 待确认 RunnableConfig      RunnableConfig                            // 运行配置
	Trace               *ConversationTrace                      // 追踪记录
	Channel             chan string                             // 响应流
	LeaseKey            string                                  // 租约锁键
	answerBuffer        strings.Builder                         // 响应内容缓冲区
	mu                  sync.Mutex                              // 响应内容缓冲区锁
	ThinkingSteps       *list.CopyOnWriteList[string]           // 思考步骤列表
	references          *list.CopyOnWriteList[*SearchReference] // 引用列表
	usedTools           *list.CopyOnWriteList[string]           // 已使用的工具集合
	StartTime           time.Time                               // 开始时间（毫秒精度）
	FirstResponseTimeMs atomic.Int64                            // 首次响应耗时（毫秒）
	Finalized           atomic.Bool                             // 是否已完成
	CancelFunc          context.CancelFunc                      // 资源释放
}

func NewConversationContext(plan *StreamLaunchPlan) *ConversationContext {
	return &ConversationContext{
		ConversationId:       plan.ConversationId,
		Question:             plan.Question,
		ChatMode:             plan.ChatMode,
		SelectedDocumentId:   plan.SelectedDocumentId,
		SelectedDocumentName: plan.SelectedDocumentName,
		SelectedTaskId:       plan.SelectedTaskId,
		CurrentDate:          plan.CurrentDate,
		CurrentDateText:      plan.CurrentDateText,
		ThinkingSteps:        list.NewCopyOnWriteList[string](nil),
		references:           list.NewCopyOnWriteList[*SearchReference](nil),
		usedTools:            list.NewCopyOnWriteList[string](nil),
		StartTime:            time.Now(),
	}
}

// ReleaseResources 释放资源
func (c *ConversationContext) ReleaseResources() {
	cancelFunc := c.CancelFunc
	if cancelFunc != nil {
		cancelFunc()
		c.CancelFunc = nil
	}
}

// AddThinkingSteps 添加思考步骤
func (c *ConversationContext) AddThinkingSteps(steps ...string) {
	c.ThinkingSteps.AddAll(steps)
}

// AddReferences 添加引用
func (c *ConversationContext) AddReferences(refs ...*SearchReference) {
	c.references.AddAll(refs)
}

// AddUsedTools 添加已使用的工具
func (c *ConversationContext) AddUsedTools(tools ...string) {
	for _, tool := range tools {
		if !c.usedTools.Contain(tool) && strutil.IsNotBlank(tool) {
			c.usedTools.Add(tool)
		}
	}
}

// SnapshotUsedTools 获取已使用的工具列表的快照
func (c *ConversationContext) SnapshotUsedTools() []string {
	return c.usedTools.SubList(0, c.usedTools.Size())
}

// UniqueReferences 获取唯一引用列表
func (c *ConversationContext) UniqueReferences() []*SearchReference {
	size := c.references.Size()
	if size == 0 {
		return nil
	}
	references := c.references.SubList(0, size)
	return utils.Distinct(references, func(ref *SearchReference) string {
		return ref.UniqueKey()
	})
}

// SnapshotThinkingSteps 获取思考步骤列表的快照
func (c *ConversationContext) SnapshotThinkingSteps() []string {
	size := c.ThinkingSteps.Size()
	if size == 0 {
		return nil
	}
	return c.ThinkingSteps.SubList(0, size)
}

// WriteAnswerBuffer 写入响应内容缓冲区
func (c *ConversationContext) WriteAnswerBuffer(content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.answerBuffer.WriteString(content)
}

// Answer 获取响应内容缓冲区内容
func (c *ConversationContext) Answer() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.answerBuffer.String()
}

// AnswerLength 获取响应内容缓冲区长度（字符数）
func (c *ConversationContext) AnswerLength() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return utf8.RuneCountInString(c.answerBuffer.String())
}

// ExecutionModeName 获取执行模式名称
func (c *ConversationContext) ExecutionModeName() string {
	if execPlan := c.ExecutionPlan.Load(); execPlan != nil {
		return execPlan.ExecutionModeName()
	}
	return ""
}

// NeedClarification 是否需要澄清
func (c *ConversationContext) NeedClarification() bool {
	if execPlan := c.ExecutionPlan.Load(); execPlan != nil {
		return execPlan.Mode == ExecutionModeClarification && len(execPlan.ClarificationOptions) > 0
	}
	return false
}

// ClarificationOptions 获取澄清选项
func (c *ConversationContext) ClarificationOptions() []string {
	if execPlan := c.ExecutionPlan.Load(); execPlan != nil {
		return execPlan.ClarificationOptions
	}
	return nil
}

// DebugTraceJSON 序列化调试轨迹
func (c *ConversationContext) DebugTraceJSON() string {
	dt := c.DebugTrace.Load()
	if dt == nil {
		return ""
	}
	data, err := json.Marshal(dt)
	if err != nil {
		return ""
	}
	return string(data)
}
