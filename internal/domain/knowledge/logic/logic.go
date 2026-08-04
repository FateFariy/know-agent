package logic

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
)

// KnowledgeLogic 知识管理服务接口
type KnowledgeLogic interface {
	// SaveScope 保存/更新知识范围节点
	SaveScope(ctx context.Context, scopeNode *entity.KnowledgeScopeNode) (*entity.KnowledgeScopeNode, error)

	// DeleteScope 删除知识范围节点
	DeleteScope(ctx context.Context, scopeCode string) (bool, error)

	// ListScopes 查询知识范围列表
	ListScopes(ctx context.Context) ([]*entity.KnowledgeScopeNode, error)

	// SaveTopic 保存/更新主题节点
	SaveTopic(ctx context.Context, topicNode *entity.KnowledgeTopicNode) (*entity.KnowledgeTopicNode, error)

	// DeleteTopic 删除主题节点
	DeleteTopic(ctx context.Context, topicCode string) (bool, error)

	// ListTopics 查询主题列表（支持按 scopeCode 过滤）
	ListTopics(ctx context.Context, scopeCode string) ([]*entity.KnowledgeTopicNode, error)

	// ListTopicDocumentRelations 查询主题文档关联
	ListTopicDocumentRelations(ctx context.Context, topicCode string) ([]*entity.KnowledgeTopicDocumentRelation, error)

	// SaveTopicDocumentRelation 保存主题文档关联
	SaveTopicDocumentRelation(ctx context.Context, relation *entity.KnowledgeTopicDocumentRelation) (*entity.KnowledgeTopicDocumentRelation, error)

	// RemoveTopicDocumentRelation 移除主题文档关联
	RemoveTopicDocumentRelation(ctx context.Context, topicCode string, documentId int64) (bool, error)

	// QueryRouteTracePage 分页查询知识路由追踪
	QueryRouteTracePage(ctx context.Context, conversationId, mode string, routeStatus, pageNo, pageSize int) ([]*entity.KnowledgeRouteTrace, int64, error)
}
