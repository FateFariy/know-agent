package model

import (
	"context"
	"github.com/cloudwego/eino/components/model"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type ChatModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string, opts ...model.Option) (string, error)

	StreamWithTrace(ctx context.Context, stage, systemPrompt, userPrompt string, trace *vo.ConversationTrace, opts ...model.Option) (<-chan string, error)
}
