package logic

import (
	"context"

	"github.com/duke-git/lancet/v2/slice"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// StructureNodeLogicImpl 文档结构节点服务，负责将解析/结构抽取后的候选节点转化为可持久化的结构节点实体
type StructureNodeLogicImpl struct {
	repo adapter.DocumentRepository
}

var _ StructureNodeLogic = (*StructureNodeLogicImpl)(nil)

func NewStructureNodeLogicImpl(repo adapter.DocumentRepository) *StructureNodeLogicImpl {
	return &StructureNodeLogicImpl{repo: repo}
}

// ReplaceDocumentNodes 替换文档结构节点：
func (l *StructureNodeLogicImpl) ReplaceDocumentNodes(ctx context.Context, documentId, parseTaskId int64,
	candidates []*vo.DocumentStructureNodeCandidate) ([]*entity.DocumentStructureNode, error) {
	if documentId == 0 || parseTaskId == 0 || len(candidates) == 0 {
		return nil, nil
	}

	// 按文档ID清除旧的结构节点
	if err := l.repo.DeleteStructureNodeByDocumentId(ctx, documentId); err != nil {
		return nil, err
	}

	// 过滤掉无效的候选节点
	candidates = slice.Filter(candidates, func(index int, candidate *vo.DocumentStructureNodeCandidate) bool {
		return candidate != nil && candidate.NodeNo != 0
	})

	// 分配雪花ID，并建立 nodeNo -> id 映射，便于父子/兄弟关系回写
	nodeIdMap := utils.SliceToMapBy(candidates, func(candidate *vo.DocumentStructureNodeCandidate) (int, int64) {
		return candidate.NodeNo, utils.GetSnowflakeNextID()
	})

	// 将候选节点转换为实体节点（回填分配后的ID、父节点ID、兄弟节点ID）
	nodes := slice.Map(candidates, func(index int, candidate *vo.DocumentStructureNodeCandidate) *entity.DocumentStructureNode {
		return &entity.DocumentStructureNode{
			ID:                nodeIdMap[candidate.NodeNo],
			DocumentId:        documentId,
			ParseTaskId:       parseTaskId,
			NodeNo:            candidate.NodeNo,
			NodeType:          candidate.NodeType,
			ParentNodeId:      nodeIdMap[candidate.ParentNodeNo],
			PrevSiblingNodeId: nodeIdMap[candidate.PrevSiblingNodeNo],
			NextSiblingNodeId: nodeIdMap[candidate.NextSiblingNodeNo],
			Depth:             candidate.Depth,
			NodeCode:          candidate.NodeCode,
			Title:             candidate.Title,
			AnchorText:        candidate.AnchorText,
			CanonicalPath:     candidate.CanonicalPath,
			SectionPath:       candidate.SectionPath,
			ContentText:       candidate.ContentText,
			ItemIndex:         candidate.ItemIndex,
		}
	})

	// 批量插入
	if err := l.repo.InsertStructureNodeBatch(ctx, nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

// ListDocumentNodes 查询文档结构节点列表
func (l *StructureNodeLogicImpl) ListDocumentNodes(ctx context.Context, documentId, parseTaskId int64) ([]*entity.DocumentStructureNode, error) {
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
		return slice.Filter(list, func(index int, node *entity.DocumentStructureNode) bool {
			return node.ParseTaskId == parseTaskId
		}), nil
	}
	return list, nil
}

// DeleteByDocumentId 按文档ID删除所有结构节点
func (l *StructureNodeLogicImpl) DeleteByDocumentId(ctx context.Context, documentId int64) error {
	if documentId == 0 {
		return nil
	}
	return l.repo.DeleteStructureNodeByDocumentId(ctx, documentId)
}
