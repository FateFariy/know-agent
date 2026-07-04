package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/model/vo"
	docentity "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
)

// KnowledgeRepository 领域模型视角下的知识库仓储：提供可检索文档、父级块、知识范围/主题节点、文档画像、路由跟踪等能力
type KnowledgeRepository interface {
	// Do 运行一个事务
	Do(ctx context.Context, fn func(ctx context.Context) error) error

	// SelectRetrievableDocuments 查询可检索的文档（可选按 documentIds 过滤）
	SelectRetrievableDocuments(ctx context.Context, documentIds ...int64) ([]*vo.KnowledgeDocument, error)

	// SelectParentBlocks 根据 ID 列表查询父级块
	SelectParentBlocks(ctx context.Context, parentBlockIDs []int64) ([]*docentity.DocumentParentBlock, error)

	// SelectKnowledgeScopeNodes 获取有效的知识范围节点
	SelectKnowledgeScopeNodes(ctx context.Context) ([]*entity.KnowledgeScopeNode, error)

	// SelectKnowledgeTopicNodes 获取有效的主题节点
	SelectKnowledgeTopicNodes(ctx context.Context) ([]*entity.KnowledgeTopicNode, error)

	// SelectDocumentProfiles 获取构建成功（profileStatus=2）的文档画像
	SelectDocumentProfiles(ctx context.Context) ([]*entity.KnowledgeDocumentProfile, error)

	// SelectTopicDocumentRelations 获取主题-文档映射关系
	SelectTopicDocumentRelations(ctx context.Context) ([]*entity.KnowledgeTopicDocumentRelation, error)

	// InsertKnowledgeRouteTrace 写入一条路由跟踪记录
	InsertKnowledgeRouteTrace(ctx context.Context, trace *entity.KnowledgeRouteTrace) error
}
