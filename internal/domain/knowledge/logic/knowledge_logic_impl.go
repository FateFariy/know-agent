package logic

import (
	"context"
	"errors"
	"sort"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	documentlogic "github.com/swiftbit/know-agent/internal/domain/document/logic"
	documentvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
)

// KnowledgeLogicImpl 知识管理领域实现
type KnowledgeLogicImpl struct {
	repo          adapter.KnowledgeRepository
	documentLogic documentlogic.LifecycleLogic
}

// NewKnowledgeLogic 构造函数
func NewKnowledgeLogic(repo adapter.KnowledgeRepository, documentLogic documentlogic.LifecycleLogic) *KnowledgeLogicImpl {
	return &KnowledgeLogicImpl{repo: repo, documentLogic: documentLogic}
}

// ============ Scope ============

func (k *KnowledgeLogicImpl) SaveScope(ctx context.Context, scopeNode *entity.KnowledgeScopeNode) (*entity.KnowledgeScopeNode, error) {
	if err := k.repo.UpsertKnowledgeScopeNode(ctx, scopeNode); err != nil {
		return nil, err
	}
	return scopeNode, nil
}

func (k *KnowledgeLogicImpl) DeleteScope(ctx context.Context, scopeCode string) (bool, error) {
	if err := k.repo.DeleteKnowledgeScopeNode(ctx, strutil.Trim(scopeCode)); err != nil {
		return false, err
	}
	return true, nil
}

func (k *KnowledgeLogicImpl) ListScopes(ctx context.Context) ([]*entity.KnowledgeScopeNode, error) {
	return k.repo.SelectKnowledgeScopeNodes(ctx)
}

// ============ Topic ============

func (k *KnowledgeLogicImpl) SaveTopic(ctx context.Context, topicNode *entity.KnowledgeTopicNode) (*entity.KnowledgeTopicNode, error) {
	if err := k.repo.UpsertKnowledgeTopicNode(ctx, topicNode); err != nil {
		return nil, err
	}
	return topicNode, nil
}

func (k *KnowledgeLogicImpl) DeleteTopic(ctx context.Context, topicCode string) (bool, error) {
	if err := k.repo.DeleteKnowledgeTopicNode(ctx, strutil.Trim(topicCode)); err != nil {
		return false, err
	}
	return true, nil
}

func (k *KnowledgeLogicImpl) ListTopics(ctx context.Context, scopeCode string) ([]*entity.KnowledgeTopicNode, error) {
	var nodes []*entity.KnowledgeTopicNode
	var err error

	if strutil.IsBlank(scopeCode) {
		nodes, err = k.repo.SelectKnowledgeTopicNodes(ctx)
	} else {
		nodes, err = k.repo.SelectKnowledgeTopicNodesByScopeCode(ctx, strutil.Trim(scopeCode))
	}
	if err != nil {
		return nil, err
	}
	sort.SliceStable(nodes, func(i, j int) bool { return nodes[i].SortOrder < nodes[j].SortOrder })
	return nodes, nil
}

// ============ Document Profile ============

func (k *KnowledgeLogicImpl) GetDocumentProfile(ctx context.Context, documentId int64) (*entity.KnowledgeDocumentProfile, error) {
	if documentId <= 0 {
		return nil, errors.New("documentId 非法")
	}
	return k.repo.SelectDocumentProfileByDocumentId(ctx, documentId)
}

func (k *KnowledgeLogicImpl) RegenerateDocumentProfile(ctx context.Context, documentId int64) (*entity.KnowledgeDocumentProfile, error) {
	if documentId <= 0 {
		return nil, errors.New("documentId 非法")
	}
	// 此处仅执行画像的写入/更新操作；真正的“重新生成”由上游的异步 pipeline 完成
	profile := &entity.KnowledgeDocumentProfile{
		DocumentId:    documentId,
		ProfileStatus: 1, // 标记为生成中
	}
	if err := k.repo.UpsertDocumentProfile(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (k *KnowledgeLogicImpl) BatchRegenerateDocumentProfiles(ctx context.Context, documentIds []int64) ([]*entity.KnowledgeDocumentProfile, error) {
	if len(documentIds) == 0 {
		return nil, errors.New("documentIds 不能为空")
	}
	profiles := make([]*entity.KnowledgeDocumentProfile, 0, len(documentIds))
	for _, id := range documentIds {
		if id <= 0 {
			continue
		}
		profiles = append(profiles, &entity.KnowledgeDocumentProfile{DocumentId: id, ProfileStatus: 1})
	}
	if err := k.repo.BatchUpsertDocumentProfiles(ctx, profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

// ============ Topic-Document Relation ============

func (k *KnowledgeLogicImpl) ListTopicDocumentRelations(ctx context.Context, topicCode string) ([]*entity.KnowledgeTopicDocumentRelation, error) {
	relations, err := k.repo.SelectTopicDocumentRelationsByTopicCode(ctx, strutil.Trim(topicCode))
	if err != nil {
		return nil, err
	}
	documents, err := k.documentLogic.ListRetrievableDocuments(ctx)
	if err != nil {
		return nil, err
	}
	docMap := utils.SliceToMapBy(documents, func(doc *documentvo.KnowledgeDocument) (int64, *documentvo.KnowledgeDocument) {
		return doc.DocumentId, doc
	})

	for _, rel := range relations {
		if doc := docMap[rel.DocumentId]; doc != nil {
			rel.DocumentName = doc.DocumentName
			rel.KnowledgeScopeCode = doc.KnowledgeScopeCode
			rel.KnowledgeScopeName = doc.KnowledgeScopeName
			rel.BusinessCategory = doc.BusinessCategory
			rel.DocumentTags = doc.DocumentTags
		}
	}
	return relations, nil
}

// SaveTopicDocumentRelation 保存/更新主题-文档关系
func (k *KnowledgeLogicImpl) SaveTopicDocumentRelation(ctx context.Context, relation *entity.KnowledgeTopicDocumentRelation) (*entity.KnowledgeTopicDocumentRelation, error) {
	relation.RelationSource = utils.BlankToDefault(relation.RelationSource, "manual")
	if err := k.repo.UpsertTopicDocumentRelation(ctx, relation); err != nil {
		return nil, err
	}
	return relation, nil
}

// RemoveTopicDocumentRelation 删除主题-文档关系
func (k *KnowledgeLogicImpl) RemoveTopicDocumentRelation(ctx context.Context, topicCode string, documentId int64) (bool, error) {
	if err := k.repo.DeleteTopicDocumentRelation(ctx, strutil.Trim(topicCode), documentId); err != nil {
		return false, err
	}
	return true, nil
}

// ============ Route Trace ============

func (k *KnowledgeLogicImpl) QueryRouteTracePage(ctx context.Context, conversationId, mode string, routeStatus, pageNo, pageSize int) ([]*entity.KnowledgeRouteTrace, int64, error) {
	pageNo = max(1, pageNo)
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	return k.repo.SelectKnowledgeRouteTracePage(ctx, strutil.Trim(conversationId), strutil.Trim(mode), routeStatus, pageNo, pageSize)
}
