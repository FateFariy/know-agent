package logic

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/aggregate"
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

// KnowledgeBaseLogic 知识库服务
type KnowledgeBaseLogic interface {
	// SaveKnowledgeBase 保存/更新知识库（ID=0 时插入，否则更新）
	SaveKnowledgeBase(ctx context.Context, config *entity.KnowledgeBase) (*entity.KnowledgeBase, error)

	// DeleteKnowledgeBase 删除知识库（软删除）
	DeleteKnowledgeBase(ctx context.Context, id int64) (bool, error)

	// ListKnowledgeBases 查询所有知识库列表
	ListKnowledgeBases(ctx context.Context) ([]*entity.KnowledgeBase, error)

	// GetKnowledgeBase 根据ID查询知识库详情
	GetKnowledgeBase(ctx context.Context, id int64) (*entity.KnowledgeBase, error)

	// UpdateKnowledgeBaseSetting 更新知识库（仅更新配置JSON字段）
	UpdateKnowledgeBaseSetting(ctx context.Context, config *entity.KnowledgeBase) (*entity.KnowledgeBase, error)

	// ListEnabledKnowledgeBases 查询所有启用的知识库
	ListEnabledKnowledgeBases(ctx context.Context) ([]*entity.KnowledgeBase, error)

	// ListKnowledgeBasesByIds 根据ID列表查询知识库
	ListKnowledgeBasesByIds(ctx context.Context, ids []int64) ([]*entity.KnowledgeBase, error)

	// GetEnabledKnowledgeBase 根据ID获取启用的知识库（不存在或已停用则返回错误）
	GetEnabledKnowledgeBase(ctx context.Context, id int64) (*entity.KnowledgeBase, error)

	// ListKnowledgeBaseOptions	查询知识库选项列表（包含可检索文档数量）
	ListKnowledgeBaseOptions(ctx context.Context) ([]*KnowledgeConfigOption, error)
}

// KnowledgeBaseRetrievalLogic 知识库检索范围
type KnowledgeBaseRetrievalLogic interface {
	// Resolve 根据聊天模式和知识库选择模式解析检索范围
	Resolve(ctx context.Context, chatMode, selectionMode string, selectedKnowledgeBaseIds []string) (*aggregate.KnowledgeBaseSelectionSnapshot, error)
}

// KnowledgeConfigOption 知识库选项（用于下拉选择等场景）
type KnowledgeConfigOption struct {
	ID               int64  // 知识库ID
	BaseName         string // 知识库名称
	Description      string // 描述
	IsDefault        int    // 是否默认
	RetrievableCount int64  // 可检索文档数量
}
