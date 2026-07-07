package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// Executor 对话执行器接口，根据执行模式负责生成最终回答。
type Executor interface {
	// Mode 返回当前执行器对应的执行模式
	Mode() vo.ExecutionMode

	// Execute 执行回答生成逻辑
	Execute(ctx context.Context, convCtx *vo.ConversationContext) (<-chan string, error)
}

// RagPromptAssembler RAG 提示词组装接口
type RagPromptAssembler interface {
	Assemble(ctx context.Context, plan *vo.ConversationExecutionPlan, retrievalCtx *vo.RagRetrievalContext) (*vo.RagPromptAssemblyResult, error)
}
