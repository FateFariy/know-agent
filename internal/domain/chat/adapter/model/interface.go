package model

import (
	"context"
)

type ChatModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string, opts ...Option) (string, error)

	StreamWithTrace(ctx context.Context, stage, systemPrompt, userPrompt string, opts ...Option) (<-chan string, error)
}
