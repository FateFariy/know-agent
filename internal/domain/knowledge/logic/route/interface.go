package route

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/rank"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// KnowledgeRouter 知识路由器
type KnowledgeRouter interface {
	// Route 根据问题进行知识路由
	Route(ctx context.Context, question, rewriteQuestion string) (*vo.KnowledgeRouteDecision, error)

	// RecordAutoRoute 记录自动路由结果
	RecordAutoRoute(ctx context.Context, exchangeId int64, conversationId, question, rewriteQuestion string, decision *vo.KnowledgeRouteDecision) error

	// RecordShadowRoute 记录影子路由结果
	RecordShadowRoute(ctx context.Context, exchangeId, documentId int64, conversationId, question, rewriteQuestion string) error
}

// Ranker 统一排名器
type Ranker interface {
	// Order() int
	Rank(ctx context.Context, rankCtx *rank.Context) error
}
