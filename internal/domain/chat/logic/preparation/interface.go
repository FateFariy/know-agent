package preparation

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ConversationPreOrchestrator 聊天前置编排器接口
type ConversationPreOrchestrator interface {
	// Prepare 准备对话执行计划
	Prepare(ctx context.Context, convCtx *conversation.Context) (*vo.ConversationExecutionPlan, error)
}
