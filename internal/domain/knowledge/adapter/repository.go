package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
)

// KnowledgeRepository 领域模型视角下的知识库仓储：提供可检索文档、父级块、知识范围/主题节点、文档画像、路由跟踪等能力
type KnowledgeRepository interface {
	// Do 运行一个事务
	Do(ctx context.Context, fn func(ctx context.Context) error) error

	// ========== 知识范围节点相关 ==========

	// SelectKnowledgeScopeNodes 获取有效的知识范围节点
	SelectKnowledgeScopeNodes(ctx context.Context) ([]*entity.KnowledgeScopeNode, error)

	// UpsertKnowledgeScopeNode 保存/更新知识范围节点（按 scopeCode 判重）
	UpsertKnowledgeScopeNode(ctx context.Context, node *entity.KnowledgeScopeNode) error

	// DeleteKnowledgeScopeNode 按 scopeCode 删除知识范围节点
	DeleteKnowledgeScopeNode(ctx context.Context, scopeCode string) error

	// ========== 主题节点相关 ==========

	// SelectKnowledgeTopicNodes 获取有效的主题节点
	SelectKnowledgeTopicNodes(ctx context.Context) ([]*entity.KnowledgeTopicNode, error)

	// SelectKnowledgeTopicNodesByScopeCode 按 scopeCode 过滤主题节点
	SelectKnowledgeTopicNodesByScopeCode(ctx context.Context, scopeCode string) ([]*entity.KnowledgeTopicNode, error)

	// UpsertKnowledgeTopicNode 保存/更新主题节点（按 topicCode 判重）
	UpsertKnowledgeTopicNode(ctx context.Context, node *entity.KnowledgeTopicNode) error

	// DeleteKnowledgeTopicNode 按 topicCode 删除主题节点
	DeleteKnowledgeTopicNode(ctx context.Context, topicCode string) error

	// ========== 文档画像相关 ==========

	// SelectDocumentProfiles 获取构建成功的文档画像
	SelectDocumentProfiles(ctx context.Context) ([]*entity.KnowledgeDocumentProfile, error)

	// SelectDocumentProfileByDocumentId 根据文档ID获取画像
	SelectDocumentProfileByDocumentId(ctx context.Context, documentId int64) (*entity.KnowledgeDocumentProfile, error)

	// UpsertDocumentProfile 保存/更新文档画像
	UpsertDocumentProfile(ctx context.Context, profile *entity.KnowledgeDocumentProfile) error

	// BatchUpsertDocumentProfiles 批量保存/更新文档画像
	BatchUpsertDocumentProfiles(ctx context.Context, profiles []*entity.KnowledgeDocumentProfile) error

	// ========== 主题-文档关系相关 ==========

	// SelectTopicDocumentRelations 获取主题-文档映射关系
	SelectTopicDocumentRelations(ctx context.Context) ([]*entity.KnowledgeTopicDocumentRelation, error)

	// SelectTopicDocumentRelationsByTopicCode 按主题查询关联关系
	SelectTopicDocumentRelationsByTopicCode(ctx context.Context, topicCode string) ([]*entity.KnowledgeTopicDocumentRelation, error)

	// UpsertTopicDocumentRelation 保存/更新主题-文档关联
	UpsertTopicDocumentRelation(ctx context.Context, relation *entity.KnowledgeTopicDocumentRelation) error

	// DeleteTopicDocumentRelation 按 topicCode+documentId 删除主题-文档关联
	DeleteTopicDocumentRelation(ctx context.Context, topicCode string, documentId int64) error

	// ========== 路由跟踪相关 ==========

	// InsertKnowledgeRouteTrace 写入一条路由跟踪记录
	InsertKnowledgeRouteTrace(ctx context.Context, trace *entity.KnowledgeRouteTrace) error

	// SelectKnowledgeRouteTracePage 分页查询路由跟踪记录
	SelectKnowledgeRouteTracePage(ctx context.Context, conversationId, mode string, routeStatus, pageNo, pageSize int) ([]*entity.KnowledgeRouteTrace, int64, error)
}
