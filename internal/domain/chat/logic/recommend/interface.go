package recommend

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
)

// QuestionRecommender 追问推荐器
type QuestionRecommender interface {
	// Generate 生成推荐追问
	Generate(ctx context.Context, question, answer string, recentExchanges []*entity.ChatExchange) []string
}
