package logic

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

// KnowledgeLogicImpl 知识管理领域实现
type KnowledgeLogicImpl struct {
	repo       adapter.KnowledgeRepository
	docGateway adapter.DocumentGateway
}

func NewKnowledgeLogicImpl(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway) *KnowledgeLogicImpl {
	return &KnowledgeLogicImpl{repo: repo, docGateway: docGateway}
}

// ============ Scope ============

// SaveScope 保存知识范围
func (k *KnowledgeLogicImpl) SaveScope(ctx context.Context, scopeNode *entity.KnowledgeScopeNode) (*entity.KnowledgeScopeNode, error) {
	// 验证知识库存在
	kb, err := k.repo.SelectKnowledgeBaseById(ctx, scopeNode.KnowledgeBaseId)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, errorx.ErrKnowledgeBaseNotFound.Format(scopeNode.KnowledgeBaseId)
	}

	// 验证父级范围
	if scopeNode.ParentScopeId > 0 {
		parentScope, err := k.repo.SelectScopeById(ctx, scopeNode.ParentScopeId, scopeNode.KnowledgeBaseId)
		if err != nil {
			return nil, err
		}
		if parentScope == nil {
			return nil, common.NewBizError(common.ErrInvalidParam.Code, "父级知识范围不存在或不属于当前知识库")
		}
		if scopeNode.ID > 0 && scopeNode.ID == scopeNode.ParentScopeId {
			return nil, common.NewBizError(common.ErrInvalidParam.Code, "父级知识范围不能是自己")
		}
	}

	// 如果是更新操作，验证目标范围存在
	if scopeNode.ID > 0 {
		existing, err := k.repo.SelectScopeById(ctx, scopeNode.ID, scopeNode.KnowledgeBaseId)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, common.NewBizError(common.ErrInvalidParam.Code, "知识范围不存在或不属于当前知识库")
		}
	}

	// 名称唯一性校验
	existingByName, err := k.repo.SelectScopeByName(ctx, scopeNode.KnowledgeBaseId, scopeNode.ScopeName, scopeNode.ID)
	if err != nil {
		return nil, err
	}
	if existingByName != nil {
		return nil, common.NewBizError(common.ErrInvalidParam.Code, "知识范围名称已存在")
	}

	// 新记录生成ID
	if scopeNode.ID == 0 {
		scopeNode.ID = utils.GetSnowflakeNextID()
	}

	if err = k.repo.UpsertKnowledgeScopeNode(ctx, scopeNode); err != nil {
		return nil, err
	}
	return scopeNode, nil
}

// DeleteScope 删除知识范围
func (k *KnowledgeLogicImpl) DeleteScope(ctx context.Context, id int64, kbId int64) (bool, error) {
	// 验证范围存在
	scope, err := k.repo.SelectScopeById(ctx, id, kbId)
	if err != nil {
		return false, err
	}
	if scope == nil {
		return false, common.NewBizError(common.ErrInvalidParam.Code, "知识范围不存在或不属于当前知识库")
	}

	// 检查子级范围
	childCount, err := k.repo.CountChildScopes(ctx, kbId, id)
	if err != nil {
		return false, err
	}
	if childCount > 0 {
		return false, common.NewBizError(common.ErrInvalidParam.Code, "请先删除或调整子级知识范围")
	}

	// 检查范围下的主题
	topicCount, err := k.repo.CountTopicsByScope(ctx, kbId, id)
	if err != nil {
		return false, err
	}
	if topicCount > 0 {
		return false, common.NewBizError(common.ErrInvalidParam.Code, "请先删除或调整该范围下的知识主题")
	}

	// 删除
	if err = k.repo.DeleteScope(ctx, id, kbId); err != nil {
		return false, err
	}
	return true, nil
}

// ListScopes 查询知识范围列表
func (k *KnowledgeLogicImpl) ListScopes(ctx context.Context, kbId int64) ([]*entity.KnowledgeScopeNode, error) {
	return k.repo.SelectScopesByKbId(ctx, kbId)
}

// ============ Topic ============

// SaveTopic 保存知识主题
func (k *KnowledgeLogicImpl) SaveTopic(ctx context.Context, topicNode *entity.KnowledgeTopicNode) (*entity.KnowledgeTopicNode, error) {
	// 验证知识库存在
	kb, err := k.repo.SelectKnowledgeBaseById(ctx, topicNode.KnowledgeBaseId)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, errorx.ErrKnowledgeBaseNotFound.Format(topicNode.KnowledgeBaseId)
	}

	// 验证所属范围存在
	scope, err := k.repo.SelectScopeById(ctx, topicNode.ScopeId, topicNode.KnowledgeBaseId)
	if err != nil {
		return nil, err
	}
	if scope == nil {
		return nil, common.NewBizError(common.ErrInvalidParam.Code, "所属知识范围不存在或不属于当前知识库")
	}

	// 如果是更新操作，验证目标主题存在
	if topicNode.ID > 0 {
		existing, err := k.repo.SelectTopicByID(ctx, topicNode.ID, topicNode.KnowledgeBaseId)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, common.NewBizError(common.ErrInvalidParam.Code, "知识主题不存在或不属于当前知识库")
		}
	}

	// 名称唯一性校验
	existingByName, err := k.repo.SelectTopicByName(ctx, topicNode.KnowledgeBaseId, topicNode.ScopeId, topicNode.TopicName, topicNode.ID)
	if err != nil {
		return nil, err
	}
	if existingByName != nil {
		return nil, common.NewBizError(common.ErrInvalidParam.Code, "知识主题名称已存在")
	}

	// 新记录生成ID
	if topicNode.ID == 0 {
		topicNode.ID = utils.GetSnowflakeNextID()
	}

	if err = k.repo.UpsertKnowledgeTopicNode(ctx, topicNode); err != nil {
		return nil, err
	}
	return topicNode, nil
}

// DeleteTopic 删除知识主题
func (k *KnowledgeLogicImpl) DeleteTopic(ctx context.Context, id int64, kbId int64) (bool, error) {
	// 验证主题存在
	topic, err := k.repo.SelectTopicByID(ctx, id, kbId)
	if err != nil {
		return false, err
	}
	if topic == nil {
		return false, common.NewBizError(common.ErrInvalidParam.Code, "知识主题不存在或不属于当前知识库")
	}

	// 检查主题下的文档关联
	relationCount, err := k.repo.CountRelationsByTopic(ctx, kbId, id)
	if err != nil {
		return false, err
	}
	if relationCount > 0 {
		return false, common.NewBizError(common.ErrInvalidParam.Code, "请先移除该主题下的文档关联")
	}

	// 软删除
	if err = k.repo.DeleteTopic(ctx, id, kbId); err != nil {
		return false, err
	}
	return true, nil
}

// ListTopics 查询知识主题列表
func (k *KnowledgeLogicImpl) ListTopics(ctx context.Context, kbId int64, scopeId int64) ([]*entity.KnowledgeTopicNode, error) {
	return k.repo.ListTopics(ctx, kbId, scopeId)
}

// ============ Topic-Document Relation ============

// ListTopicDocumentRelations 查询主题-文档关系列表
func (k *KnowledgeLogicImpl) ListTopicDocumentRelations(ctx context.Context, kbId int64, topicId int64) ([]*entity.KnowledgeTopicDocumentRelation, error) {
	relations, err := k.repo.ListTopicDocumentRelations(ctx, kbId, topicId)
	if err != nil {
		return nil, err
	}

	// 填充文档名称
	if len(relations) > 0 {
		docIds := make([]int64, 0, len(relations))
		for _, rel := range relations {
			docIds = append(docIds, rel.DocumentId)
		}
		documents, err := k.docGateway.FindRetrieveDocumentByIds(ctx, docIds...)
		if err != nil {
			return nil, err
		}
		docMap := utils.MapBy(documents, func(doc *vo.DocumentMetadata) (int64, *vo.DocumentMetadata) {
			return doc.DocumentId, doc
		})
		for _, rel := range relations {
			if doc := docMap[rel.DocumentId]; doc != nil {
				rel.DocumentName = doc.DocumentName
			}
		}
	}
	return relations, nil
}

// SaveTopicDocumentRelation 保存/更新主题-文档关系
func (k *KnowledgeLogicImpl) SaveTopicDocumentRelation(ctx context.Context, relation *entity.KnowledgeTopicDocumentRelation) (*entity.KnowledgeTopicDocumentRelation, error) {
	// 验证主题存在且属于知识库
	topic, err := k.repo.SelectTopicByID(ctx, relation.TopicId, relation.KnowledgeBaseId)
	if err != nil {
		return nil, err
	}
	if topic == nil {
		return nil, common.NewBizError(common.ErrInvalidParam.Code, "知识主题不存在或不属于当前知识库")
	}

	// 验证文档存在且属于知识库
	documents, err := k.docGateway.FindRetrieveDocumentByIds(ctx, relation.DocumentId)
	if err != nil {
		return nil, err
	}
	docFound := false
	for _, doc := range documents {
		if doc.DocumentId == relation.DocumentId && doc.KnowledgeBaseId == relation.KnowledgeBaseId {
			docFound = true
			break
		}
	}
	if !docFound {
		return nil, common.NewBizError(common.ErrInvalidParam.Code, "文档不属于当前知识库")
	}

	// 查找是否已存在相同关系
	existing, err := k.repo.SelectTopicDocumentRelation(ctx, relation.KnowledgeBaseId, relation.TopicId, relation.DocumentId)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// 更新现有关系
		relation.ID = existing.ID
	} else {
		// 创建新关系
		relation.ID = utils.GetSnowflakeNextID()
	}

	// 设置默认值
	relation.RelationSource = utils.BlankToDefault(relation.RelationSource, "manual")

	if err = k.repo.UpsertTopicDocumentRelation(ctx, relation); err != nil {
		return nil, err
	}
	return relation, nil
}

// RemoveTopicDocumentRelation 删除主题-文档关系
func (k *KnowledgeLogicImpl) RemoveTopicDocumentRelation(ctx context.Context, kbId int64, topicId int64, documentId int64) (bool, error) {
	// 验证关系存在
	existing, err := k.repo.SelectTopicDocumentRelation(ctx, kbId, topicId, documentId)
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, common.NewBizError(common.ErrInvalidParam.Code, "主题-文档关系不存在")
	}

	// 软删除
	if err = k.repo.DeleteTopicDocumentRelation(ctx, kbId, topicId, documentId); err != nil {
		return false, err
	}
	return true, nil
}

// ============ Route Trace ============

// QueryRouteTracePage 分页查询路由跟踪记录
func (k *KnowledgeLogicImpl) QueryRouteTracePage(ctx context.Context, conversationId, mode string, routeStatus, pageNo, pageSize int) ([]*entity.KnowledgeRouteTrace, int64, error) {
	pageNo = max(1, pageNo)
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	return k.repo.SelectKnowledgeRouteTracePage(ctx, strutil.Trim(conversationId), strutil.Trim(mode), routeStatus, pageNo, pageSize)
}
