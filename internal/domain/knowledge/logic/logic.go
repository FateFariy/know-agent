package logic

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

// KnowledgeRouteLogic 知识路由服务接口
type KnowledgeRouteLogic interface {
	// Route 根据问题进行知识路由
	Route(ctx context.Context, question, rewriteQuestion string) (*vo.KnowledgeRouteDecision, error)

	// RecordAutoRoute 记录自动路由结果
	RecordAutoRoute(ctx context.Context, exchangeId int64, conversationId, question, rewriteQuestion string, decision *vo.KnowledgeRouteDecision) error

	// RecordShadowRoute 记录影子路由结果
	RecordShadowRoute(ctx context.Context, exchangeId, documentId int64, conversationId, question, rewriteQuestion string) error
}

func Warnf(format string, v ...any) {
	logx.Alert(fmt.Sprintf(format, v...))
}
