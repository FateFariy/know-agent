package process

import (
	"context"

	"github.com/duke-git/lancet/v2/slice"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// StructureNodeManageImpl 文档结构管理服务，负责将解析/结构抽取后的候选节点转化为可持久化的结构节点实体
type StructureNodeManageImpl struct {
	repo adapter.DocumentRepository
}

var _ StructureNodeManager = (*StructureNodeManageImpl)(nil)

func NewStructureNodeManager(repo adapter.DocumentRepository) *StructureNodeManageImpl {
	return &StructureNodeManageImpl{repo: repo}
}

// ListDocumentNodes 查询文档结构节点列表
func (l *StructureNodeManageImpl) ListDocumentNodes(ctx context.Context, documentId, parseTaskId int64) ([]*entity.StructureNode, error) {
	if documentId == 0 {
		return nil, nil
	}
	// 查询文档结构节点列表
	list, err := l.repo.SelectStructureNodeListByDocumentId(ctx, documentId)
	if err != nil {
		return nil, err
	}
	// 过滤属于该任务的节点（兼容"不同任务版本"的场景）
	if parseTaskId > 0 {
		return slice.Filter(list, func(index int, node *entity.StructureNode) bool {
			return node.ParseTaskId == parseTaskId
		}), nil
	}
	return list, nil
}

// DeleteByDocumentId 按文档ID删除所有结构节点
func (l *StructureNodeManageImpl) DeleteByDocumentId(ctx context.Context, documentId int64) error {
	if documentId == 0 {
		return nil
	}
	return l.repo.DeleteStructureNodeByDocumentId(ctx, documentId)
}
