package logic

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// RetrievalChannel 检索通道接口
type RetrievalChannel interface {
	// ChannelName 检索通道名称
	ChannelName() string

	// Supports 是否支持该执行计划
	Supports(plan *vo.ConversationExecutionPlan) bool

	// Retrieve 根据子问题检索
	Retrieve(ctx context.Context, subQuestion string, plan *vo.ConversationExecutionPlan) (*RetrievalChannelResult, error)
}
