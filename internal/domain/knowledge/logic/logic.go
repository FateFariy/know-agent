package logic

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	vo2 "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
)

// KnowledgeLogic 文档知识服务
type KnowledgeLogic interface {
	// ListRetrievableDocuments 获取可检索的文档列表
	ListRetrievableDocuments(ctx context.Context) ([]*vo.KnowledgeDocument, error)

	// VectorSearch 向量检索
	VectorSearch(ctx context.Context, request *vo2.DocumentRetrieve) ([]*vo2.DocumentChunk, error)

	// KeywordSearch 关键词检索
	KeywordSearch(ctx context.Context, request *vo2.DocumentRetrieve) ([]*vo2.DocumentChunk, error)

	// ElevateToParentBlocks 将子文档提升到父块级别
	ElevateToParentBlocks(ctx context.Context, childDocuments []*vo2.DocumentChunk, maxChars int) ([]*vo2.DocumentChunk, error)
}

// KnowledgeRouteLogic 知识路由服务接口
type KnowledgeRouteLogic interface {
	// Route 根据问题进行知识路由
	Route(ctx context.Context, question, rewriteQuestion string) (*vo.KnowledgeRouteDecision, error)

	// RecordAutoRoute 记录自动路由结果
	RecordAutoRoute(ctx context.Context, exchangeId int64, conversationId, question, rewriteQuestion string, decision *vo.KnowledgeRouteDecision) error

	// RecordShadowRoute 记录影子路由结果
	RecordShadowRoute(ctx context.Context, exchangeId int64, conversationId string, documentId int64, question, rewriteQuestion string) error
}

func Warnf(format string, v ...any) {
	logx.Alert(fmt.Sprintf(format, v...))
}
