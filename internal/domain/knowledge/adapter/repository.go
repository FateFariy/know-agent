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

	// SelectKnowledgeScopeNodesByKbIds 根据知识库ID获取有效的知识范围节点
	SelectKnowledgeScopeNodesByKbIds(ctx context.Context, kbIds []int64) ([]*entity.KnowledgeScopeNode, error)

	// UpsertKnowledgeScopeNode 保存/更新知识范围节点（按 scopeCode 判重）
	UpsertKnowledgeScopeNode(ctx context.Context, node *entity.KnowledgeScopeNode) error

	// DeleteKnowledgeScopeNode 按 scopeCode 删除知识范围节点
	DeleteKnowledgeScopeNode(ctx context.Context, scopeCode string) error

	// ========== 主题节点相关 ==========

	// SelectKnowledgeTopicNodesByKbIds 根据知识库ID获取有效的主题节点
	SelectKnowledgeTopicNodesByKbIds(ctx context.Context, kbIds []int64) ([]*entity.KnowledgeTopicNode, error)

	// SelectKnowledgeTopicNodesByScopeCode 按 scopeCode 过滤主题节点
	SelectKnowledgeTopicNodesByScopeId(ctx context.Context, scopeId int64) ([]*entity.KnowledgeTopicNode, error)

	// UpsertKnowledgeTopicNode 保存/更新主题节点（按 topicCode 判重）
	UpsertKnowledgeTopicNode(ctx context.Context, node *entity.KnowledgeTopicNode) error

	// DeleteKnowledgeTopicNode 按 topicCode 删除主题节点
	DeleteKnowledgeTopicNode(ctx context.Context, topicCode string) error

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

	// ========== 知识库配置相关 ==========

	// InsertKnowledgeBase 插入知识库
	InsertKnowledgeBase(ctx context.Context, config *entity.KnowledgeBase) error

	// UpdateKnowledgeBaseById 根据ID更新知识库
	UpdateKnowledgeBaseById(ctx context.Context, config *entity.KnowledgeBase) error

	// DeleteKnowledgeBaseById 根据ID删除知识库
	DeleteKnowledgeBaseById(ctx context.Context, id int64) error

	// SelectKnowledgeBaseById 根据ID查询知识库
	SelectKnowledgeBaseById(ctx context.Context, id int64) (*entity.KnowledgeBase, error)

	// SelectKnowledgeBases 查询所有知识库
	SelectKnowledgeBases(ctx context.Context) ([]*entity.KnowledgeBase, error)

	// SelectKnowledgeBaseByIds 根据ID列表查询知识库
	SelectKnowledgeBaseByIds(ctx context.Context, ids []int64) ([]*entity.KnowledgeBase, error)

	// SelectKnowledgeBaseByBaseName 根据名称查询知识库
	SelectKnowledgeBaseByBaseName(ctx context.Context, baseName string) (*entity.KnowledgeBase, error)

	// ClearOtherDefaults 清除其他知识库的默认标记
	ClearOtherDefaults(ctx context.Context, currentId int64) error
}
