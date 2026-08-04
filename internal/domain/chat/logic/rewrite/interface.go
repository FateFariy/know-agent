package rewrite

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

type QueryRewriter interface {
	Rewrite(ctx context.Context, question, historySummary string) (*vo.QuestionRewriteResult, error)
}
