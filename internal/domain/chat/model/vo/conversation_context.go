package vo

import (
	"sync/atomic"
	"time"
)

type StreamEventMetadata struct {
	ConversationId string
	ExchangeId     int64
}

type ConversationContext struct {
	ConversationId         string                                    // 对话ID
	ExchangeId             int64                                     // 交换ID
	Question               string                                    // 用户问题
	ChatMode               ChatQueryMode                             // 聊天模式
	TraceId                string                                    // 追踪ID
	executionPlan          atomic.Pointer[ConversationExecutionPlan] // 执行计划（对应volatile）
	debugTrace             atomic.Pointer[ChatDebugTrace]            // 调试追踪（对应volatile）
	RunnableConfig         RunnableConfig                            // 运行配置
	TraceRecorder          ConversationTraceRecorder                 // 追踪记录器
	Sink                   chan string                               // 响应流（对应Sinks.Many<String>，用channel模拟）
	EventMetadata          StreamEventMetadata                       // 流式事件元数据
	AnswerBuffer           []string                                  // 响应内容缓冲区（原StringBuffer，改为切片）
	ThinkingSteps          []string                                  // 思考步骤列表
	References             []SearchReference                         // 引用列表
	UsedTools              map[string]struct{}                       // 已使用的工具集合（Set模拟）
	StartTime              time.Time                                 // 开始时间（毫秒精度）
	FirstResponseTimeMs    atomic.Int64                              // 首次响应耗时（毫秒）
	finalized              atomic.Bool                               // 是否已完成（对应AtomicBoolean）
	disposable             Disposable                                // 资源释放器（对应Disposable）
	leaseRenewalDisposable Disposable                                // 租约续期资源释放器
}
