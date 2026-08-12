package logic

import (
	"context"
	"strconv"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

// KnowledgeBaseRetrievalScopeServiceImpl 知识库检索范围服务实现
type KnowledgeBaseRetrievalScopeServiceImpl struct {
	repo       adapter.KnowledgeRepository
	docGateway adapter.DocumentGateway
}

// NewKnowledgeBaseRetrievalScopeServiceImpl 创建知识库检索范围服务实例
func NewKnowledgeBaseRetrievalScopeServiceImpl(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway) *KnowledgeBaseRetrievalScopeServiceImpl {
	return &KnowledgeBaseRetrievalScopeServiceImpl{
		repo:       repo,
		docGateway: docGateway,
	}
}

// Resolve 根据聊天模式和知识库选择模式解析检索范围
// 对应 Java resolve() 方法
func (s *KnowledgeBaseRetrievalScopeServiceImpl) Resolve(ctx context.Context, chatMode string, selectionMode string, selectedKnowledgeBaseIds []string) (*entity.KnowledgeBaseSelectionSnapshot, error) {
	// 解析选择模式
	resolvedMode := utils.BlankToDefault(selectionMode, enum.KbSelectionModeNone)

	snapshot := &entity.KnowledgeBaseSelectionSnapshot{
		SelectionMode: resolvedMode,
	}
	// 开放式聊天或无选择模式时返回空快照
	if chatMode == "open_chat" || resolvedMode == enum.KbSelectionModeNone {
		return snapshot, nil
	}

	// 根据模式选择知识库
	var selectedBases []*entity.KnowledgeBase
	var err error
	switch resolvedMode {
	case enum.KbSelectionModeAll:
		selectedBases, err = s.selectAllWithRetrievableDocuments(ctx)
	case enum.KbSelectionModeSelected:
		selectedBases, err = s.selectExplicit(ctx, selectedKnowledgeBaseIds)
	default:
		selectedBases = nil
	}
	if err != nil {
		return nil, err
	}

	if len(selectedBases) == 0 {
		return snapshot, nil
	}

	baseIds := utils.Map(selectedBases, func(base *entity.KnowledgeBase) int64 {
		return base.ID
	})

	// 查询允许的文档
	allowedDocuments, err := s.docGateway.FindRetrievableByKbIds(ctx, baseIds)
	if err != nil {
		return nil, err
	}

	// 提取有可检索文档的知识库
	mapBy := utils.MapBy(allowedDocuments, func(doc *vo.DocumentMetadata) (int64, struct{}) {
		return doc.KnowledgeBaseId, struct{}{}
	})

	// 收集选中的知识库ID
	selectedBaseIds := make([]int64, 0, len(selectedBases))
	selectedBaseNames := make([]string, 0, len(selectedBases))
	for _, base := range selectedBases {
		if _, exists := mapBy[base.ID]; exists {
			selectedBaseIds = append(selectedBaseIds, base.ID)
			selectedBaseNames = append(selectedBaseNames, base.BaseName)
		}
	}

	// 构建快照
	snapshot.SelectedKnowledgeBaseIds = selectedBaseIds
	snapshot.SelectedKnowledgeBaseNames = selectedBaseNames
	snapshot.SelectedKnowledgeBases = selectedBases
	snapshot.AllowedDocuments = allowedDocuments

	// 提取允许的文档ID和任务ID
	snapshot.AllowedDocumentIds = utils.FilterMapUniqueLimit(allowedDocuments, -1, func(doc *vo.DocumentMetadata) (int64, int64, bool) {
		return doc.DocumentId, doc.DocumentId, doc.DocumentId > 0
	})
	snapshot.AllowedTaskIds = utils.FilterMapUniqueLimit(allowedDocuments, -1, func(doc *vo.DocumentMetadata) (int64, int64, bool) {
		return doc.LastIndexTaskId, doc.LastIndexTaskId, doc.LastIndexTaskId > 0
	})

	return snapshot, nil
}

// selectAllWithRetrievableDocuments 选择所有启用且有可检索文档的知识库
func (s *KnowledgeBaseRetrievalScopeServiceImpl) selectAllWithRetrievableDocuments(ctx context.Context) ([]*entity.KnowledgeBase, error) {
	enabledBases, err := s.repo.SelectKnowledgeBases(ctx)
	if err != nil {
		return nil, err
	}
	return enabledBases, nil
}

// selectExplicit 根据用户选择的ID列表选择知识库
func (s *KnowledgeBaseRetrievalScopeServiceImpl) selectExplicit(ctx context.Context, selectedKnowledgeBaseIds []string) ([]*entity.KnowledgeBase, error) {
	// 解析并去重知识库ID
	ids := utils.FilterMapUniqueLimit(selectedKnowledgeBaseIds, -1, func(rawId string) (int64, int64, bool) {
		id, _ := strconv.ParseInt(rawId, 10, 64)
		return id, id, id > 0
	})

	if len(ids) == 0 {
		return nil, errorx.ErrKnowledgeBaseMissing
	}

	// 查询知识库列表
	bases, err := s.repo.SelectKnowledgeBaseByIds(ctx, ids)
	if err != nil {
		return nil, err
	}

	// 构建ID索引
	byId := utils.MapBy(bases, func(base *entity.KnowledgeBase) (int64, *entity.KnowledgeBase) {
		return base.ID, base
	})

	// 检查所有请求的ID是否都存在
	for _, id := range ids {
		if _, exists := byId[id]; !exists {
			return nil, errorx.ErrKnowledgeBaseDisabled.Format(id)
		}
	}

	// 按原始ID顺序返回
	result := make([]*entity.KnowledgeBase, 0, len(ids))
	for _, id := range ids {
		result = append(result, byId[id])
	}

	return result, nil
}
