package model

import (
	"context"

	"github.com/swiftbit/know-agent/common"
)

type ChatModel interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string, opts ...common.Option) (string, error)
}
