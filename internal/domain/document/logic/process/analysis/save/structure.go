package save

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// StructurePersistPhase 结构节点持久化阶段
type StructurePersistPhase struct {
	repo adapter.DocumentRepository
}

// NewStructurePersistPhase 创建结构节点持久化阶段
func NewStructurePersistPhase(repo adapter.DocumentRepository) *StructurePersistPhase {
	return &StructurePersistPhase{
		repo: repo,
	}
}

// Name 阶段名称
func (p *StructurePersistPhase) Name() string {
	return "结构节点阶段"
}

// Execute 执行结构节点持久化，文档结构是当前解析结果的完整快照，不与旧 task 的节点混合
func (p *StructurePersistPhase) Execute(ctx context.Context, saveCtx *Context) error {
	if saveCtx == nil || saveCtx.DocumentId == 0 || saveCtx.TaskId == 0 {
		return nil
	}

	candidates := saveCtx.AnalysisResult.StructureNodes
	if len(candidates) == 0 {
		logx.Infof("结构节点阶段跳过，无候选节点，documentId=%d", saveCtx.DocumentId)
		return nil
	}

	// 替换文档结构节点：先按文档ID删除，再按候选节点批量插入
	nodes, err := p.replaceDocumentNodes(ctx, saveCtx.DocumentId, saveCtx.TaskId, candidates)
	if err != nil {
		return err
	}
	saveCtx.StructureNodes = nodes

	return nil
}

// replaceDocumentNodes 替换文档结构节点：先按文档ID删除，再按候选节点批量插入
func (p *StructurePersistPhase) replaceDocumentNodes(ctx context.Context, documentId, parseTaskId int64,
	candidates []*entity.StructureNode) ([]*entity.StructureNode, error) {
	// 按文档ID清除旧的结构节点
	if err := p.repo.DeleteStructureNodeByDocumentId(ctx, documentId); err != nil {
		return nil, err
	}

	// 分配节点ID，并建立 nodeNo -> id 映射，便于父子/兄弟关系回写
	nodeIdMap := make(map[int]int64, len(candidates))
	for _, candidate := range candidates {
		if candidate != nil && candidate.NodeNo != 0 {
			nodeIdMap[candidate.NodeNo] = utils.GetSnowflakeNextID()
		}
	}

	// 回写节点ID、父节点ID、兄弟节点ID
	nodes := make([]*entity.StructureNode, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate != nil && candidate.NodeNo != 0 {
			candidate.ID = nodeIdMap[candidate.NodeNo]
			candidate.DocumentId = documentId
			candidate.ParseTaskId = parseTaskId
			candidate.ParentNodeId = nodeIdMap[candidate.ParentNodeNo]
			candidate.PrevSiblingNodeId = nodeIdMap[candidate.PrevSiblingNodeNo]
			candidate.NextSiblingNodeId = nodeIdMap[candidate.NextSiblingNodeNo]
			nodes = append(nodes, candidate)
		}
	}

	// 批量插入
	if err := p.repo.InsertStructureNodeBatch(ctx, nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}
