package model

import (
	"context"

	"github.com/swiftbit/know-agent/common"
)

type ChatModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string, opts ...common.Option) (string, error)

	GenerateWithTrace(ctx context.Context, stage, systemPrompt, userPrompt string, opts ...common.Option) (string, error)

	Stream(ctx context.Context, stage, systemPrompt, userPrompt string, opts ...common.Option) (<-chan string, error)
}
