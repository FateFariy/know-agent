package logic

import (
	"context"
	"errors"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

// KnowledgeBaseLogicImpl 知识库管理
type KnowledgeBaseLogicImpl struct {
	repo       adapter.KnowledgeRepository
	docGateway adapter.DocumentGateway
}

func NewKnowledgeBaseLogicImpl(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway) *KnowledgeBaseLogicImpl {
	return &KnowledgeBaseLogicImpl{
		repo:       repo,
		docGateway: docGateway,
	}
}

// SaveKnowledgeBase 保存/更新知识库
func (k *KnowledgeBaseLogicImpl) SaveKnowledgeBase(ctx context.Context, base *entity.KnowledgeBase) (*entity.KnowledgeBase, error) {
	if strutil.IsBlank(base.BaseName) {
		return nil, errors.New("baseName 不能为空")
	}

	// 名称唯一性校验
	existing, err := k.repo.SelectKnowledgeBaseByBaseName(ctx, strutil.Trim(base.BaseName))
	if err != nil && !errors.Is(err, errorx.ErrKnowledgeBaseNotFound) {
		return nil, err
	}
	if existing.ID != base.ID {
		return nil, errors.New("知识库名称已存在")
	}

	if base.ID > 0 {
		// 更新：查询现有记录确认存在
		_, err = k.repo.SelectKnowledgeBaseById(ctx, base.ID)
		if err != nil {
			return nil, err
		}
		if err = k.repo.UpdateKnowledgeBaseById(ctx, base); err != nil {
			return nil, err
		}
	} else {
		// 新增：分配雪花ID
		base.ID = utils.GetSnowflakeNextID()
		if err = k.repo.InsertKnowledgeBase(ctx, base); err != nil {
			return nil, err
		}
	}

	// 如果标记为默认，清除其他默认标记
	if base.IsDefault == utils.Pointer(1) {
		if err = k.repo.ClearOtherDefaults(ctx, base.ID); err != nil {
			return nil, err
		}
	}

	return base, nil
}

// DeleteKnowledgeBase 删除知识库
func (k *KnowledgeBaseLogicImpl) DeleteKnowledgeBase(ctx context.Context, id int64) (bool, error) {
	if id <= 0 {
		return false, errors.New("id 不能为空")
	}
	if err := k.repo.DeleteKnowledgeBaseById(ctx, id); err != nil {
		return false, err
	}
	return true, nil
}

// ListKnowledgeBases 查询所有知识库列表
func (k *KnowledgeBaseLogicImpl) ListKnowledgeBases(ctx context.Context) ([]*entity.KnowledgeBase, error) {
	return k.repo.SelectKnowledgeBases(ctx)
}

// GetKnowledgeBase 根据ID查询知识库详情
func (k *KnowledgeBaseLogicImpl) GetKnowledgeBase(ctx context.Context, id int64) (*entity.KnowledgeBase, error) {
	if id <= 0 {
		return nil, errors.New("id 不能为空")
	}
	base, err := k.repo.SelectKnowledgeBaseById(ctx, id)
	if err != nil {
		return nil, err
	}
	return base, nil
}

// UpdateKnowledgeBaseSetting 更新知识库
func (k *KnowledgeBaseLogicImpl) UpdateKnowledgeBaseSetting(ctx context.Context, config *entity.KnowledgeBase) (*entity.KnowledgeBase, error) {
	if config.ID <= 0 {
		return nil, errors.New("id 不能为空")
	}

	// 查询现有记录确认存在
	existing, err := k.repo.SelectKnowledgeBaseById(ctx, config.ID)
	if err != nil {
		return nil, err
	}

	// 仅更新配置JSON字段
	existing.RetrievalConfigJson = config.RetrievalConfigJson
	existing.GraphRagConfigJson = config.GraphRagConfigJson
	existing.RaptorConfigJson = config.RaptorConfigJson
	existing.MetadataFilterJson = config.MetadataFilterJson

	if err = k.repo.UpdateKnowledgeBaseById(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

// ListEnabledKnowledgeBases 查询所有启用的知识库
func (k *KnowledgeBaseLogicImpl) ListEnabledKnowledgeBases(ctx context.Context) ([]*entity.KnowledgeBase, error) {
	return k.repo.SelectKnowledgeBases(ctx)
}

// ListKnowledgeBasesByIds 根据ID列表查询知识库
func (k *KnowledgeBaseLogicImpl) ListKnowledgeBasesByIds(ctx context.Context, ids []int64) ([]*entity.KnowledgeBase, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return k.repo.SelectKnowledgeBaseByIds(ctx, ids)
}

// GetEnabledKnowledgeBase 根据ID获取启用的知识库
func (k *KnowledgeBaseLogicImpl) GetEnabledKnowledgeBase(ctx context.Context, id int64) (*entity.KnowledgeBase, error) {
	if id <= 0 {
		return nil, errors.New("id 不能为空")
	}
	base, err := k.repo.SelectKnowledgeBaseById(ctx, id)
	if err != nil {
		return nil, err
	}
	return base, nil
}

// ListKnowledgeBaseOptions 查询知识库选项列表
func (k *KnowledgeBaseLogicImpl) ListKnowledgeBaseOptions(ctx context.Context) ([]*KnowledgeConfigOption, error) {
	bases, err := k.repo.SelectKnowledgeBases(ctx)
	if err != nil {
		return nil, err
	}
	if len(bases) == 0 {
		return []*KnowledgeConfigOption{}, nil
	}

	// 收集所有知识库ID
	kbIds := make([]int64, 0, len(bases))
	for _, c := range bases {
		kbIds = append(kbIds, c.ID)
	}

	// 批量统计可检索文档数量
	retrievableCounts, err := k.docGateway.CountRetrievableDocumentsByKbIds(ctx, kbIds)
	if err != nil {
		return nil, err
	}

	// 构建选项列表
	options := make([]*KnowledgeConfigOption, 0, len(bases))
	for _, c := range bases {
		option := &KnowledgeConfigOption{
			ID:               c.ID,
			BaseName:         c.BaseName,
			Description:      c.Description,
			IsDefault:        utils.PointerOrDefault(c.IsDefault, 0),
			RetrievableCount: retrievableCounts[c.ID],
		}
		options = append(options, option)
	}
	return options, nil
}
