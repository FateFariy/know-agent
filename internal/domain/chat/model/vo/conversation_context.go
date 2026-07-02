package vo

import (
	"context"
	"strings"
	"sync/atomic"
	"time"
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
	Trace               *ConversationTrace  // 追踪记录
	Channel             chan string         // 响应流
	LeaseKey            string              // 租约锁键
	AnswerBuffer        strings.Builder     // 响应内容缓冲区
	ThinkingSteps       []string            // 思考步骤列表
	References          []SearchReference   // 引用列表
	UsedTools           map[string]struct{} // 已使用的工具集合
	StartTime           time.Time           // 开始时间（毫秒精度）
	FirstResponseTimeMs atomic.Int64        // 首次响应耗时（毫秒）
	Finalized           atomic.Bool         // 是否已完成
	CancelExecute       context.CancelFunc  // 资源释放
}
