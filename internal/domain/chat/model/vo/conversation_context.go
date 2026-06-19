package vo

import (
	"context"
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
	ExecutionPlan        atomic.Pointer[ConversationExecutionPlan] // 执行计划（对应volatile）
	DebugTrace           atomic.Pointer[ChatDebugTrace]            // 调试追踪（对应volatile）
	// todo 待确认 RunnableConfig      RunnableConfig                            // 运行配置
	Tracer   *ConversationTrace // 追踪记录器
	Channel  chan string        // 响应流
	LeaseKey string             // 租约锁键
	// EventMetadata       *StreamEventMetadata // 流式事件元数据
	AnswerBuffer        []string            // 响应内容缓冲区（原StringBuffer，改为切片）
	ThinkingSteps       []string            // 思考步骤列表
	References          []SearchReference   // 引用列表
	UsedTools           map[string]struct{} // 已使用的工具集合
	StartTime           time.Time           // 开始时间（毫秒精度）
	FirstResponseTimeMs atomic.Int64        // 首次响应耗时（毫秒）
	Finalized           atomic.Bool         // 是否已完成
	// disposable             Disposable                                // 资源释放器（对应Disposable）
	CancelLeaseRenewal context.CancelFunc // 租约锁取消函数
}

func (c *ConversationContext) IsFinalized() bool {
	return c.Finalized.Load()
}
