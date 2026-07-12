package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// DocumentRepository 文档数据访问接口
type DocumentRepository interface {
	// Do 运行一个事务性操作
	Do(ctx context.Context, fn func(txCtx context.Context) error) error

	// ========== 文档相关 ==========

	// InsertDocument 插入文档
	InsertDocument(ctx context.Context, document *entity.Document) error

	// SelectDocumentPage 分页查询文档
	SelectDocumentPage(ctx context.Context, pageNo, pageSize int, keyword string) ([]*entity.Document, int64, error)

	// SelectDocumentById 根据ID查询文档
	SelectDocumentById(ctx context.Context, documentId int64) (*entity.Document, error)

	// UpdateDocumentById 根据ID更新文档
	UpdateDocumentById(ctx context.Context, document *entity.Document) error

	// DeleteDocumentRelatedDataById 删除文档关联数据
	DeleteDocumentRelatedDataById(ctx context.Context, documentId int64) (string, error)

	// SelectRetrievableDocuments 查询可检索的文档
	SelectRetrievableDocuments(ctx context.Context, documentIds ...int64) ([]*vo.KnowledgeDocument, error)

	// ========== 任务相关 ==========

	// InsertTask 插入任务
	InsertTask(ctx context.Context, task *entity.DocumentTask) error

	// UpdateTaskById 根据任务ID更新任务
	UpdateTaskById(ctx context.Context, task *entity.DocumentTask) error

	// DeleteTaskByDocumentId 根据文档ID删除任务
	DeleteTaskByDocumentId(ctx context.Context, documentId int64) error

	// SelectTaskById 根据ID查询任务
	SelectTaskById(ctx context.Context, taskId int64) (*entity.DocumentTask, error)

	// SelectLatestTask 查询最新任务
	SelectLatestTask(ctx context.Context, documentId int64, taskTypes ...int) (*entity.DocumentTask, error)

	// SelectTaskListByDocumentIds 根据文档ID列表查询任务
	SelectTaskListByDocumentIds(ctx context.Context, documentIds []int64) ([]*entity.DocumentTask, error)

	// CountActiveTask 统计活跃任务数
	CountActiveTask(ctx context.Context, documentId int64, taskType int, taskStatus ...int) (int64, error)

	// ========== 任务日志相关 ==========

	// InsertTaskLog 插入任务日志
	InsertTaskLog(ctx context.Context, log *entity.DocumentTaskLog) error

	// DeleteTaskLogByDocumentId 根据文档ID删除任务日志
	DeleteTaskLogByDocumentId(ctx context.Context, documentId int64) error

	// SelectTaskLogPage 分页查询任务日志
	SelectTaskLogPage(ctx context.Context, taskId int64, pageNo, pageSize int) ([]*entity.DocumentTaskLog, int64, error)

	// ========== 方案/策略相关 ==========

	// InsertPlan 插入方案
	InsertPlan(ctx context.Context, plan *entity.DocumentStrategyPlan) error

	// UpdatePlanById 根据ID更新方案
	UpdatePlanById(ctx context.Context, plan *entity.DocumentStrategyPlan) error

	// DeletePlanByDocumentId 根据文档ID删除方案
	DeletePlanByDocumentId(ctx context.Context, documentId int64) error

	// SelectPlanById 根据ID查询方案
	SelectPlanById(ctx context.Context, planId int64) (*entity.DocumentStrategyPlan, error)

	// SelectLatestPlanVersion 查询最新方案版本
	SelectLatestPlanVersion(ctx context.Context, documentId int64) (int, error)

	// ========== 步骤相关 ==========

	// InsertStepBatch 批量插入步骤
	InsertStepBatch(ctx context.Context, steps []*entity.DocumentStrategyStep) error

	// DeleteStepByDocumentId 根据文档ID删除步骤
	DeleteStepByDocumentId(ctx context.Context, documentId int64) error

	// SelectStepListByPlanId 根据方案ID查询步骤列表
	SelectStepListByPlanId(ctx context.Context, planId int64) ([]*entity.DocumentStrategyStep, error)

	// UpdateStepExecuteStatus 更新步骤执行状态
	UpdateStepExecuteStatus(ctx context.Context, planId int64, status int) error

	// ========== 块相关 ==========

	// InsertChunk 插入块
	InsertChunk(ctx context.Context, chunk *entity.DocumentChunk) error

	// InsertChunkBatch 批量插入块
	InsertChunkBatch(ctx context.Context, chunks []*entity.DocumentChunk) error

	// UpdateChunkByTaskId 根据任务ID更新块
	UpdateChunkByTaskId(ctx context.Context, chunk *entity.DocumentChunk) error

	// DeleteChunkByDocumentId 根据文档ID删除块
	DeleteChunkByDocumentId(ctx context.Context, documentId int64) error

	// SelectChunkPage 分页查询块
	SelectChunkPage(ctx context.Context, documentId, taskId int64, pageNo, pageSize int) ([]*entity.DocumentChunk, int64, error)

	// SelectChunkById 根据ID查询块
	SelectChunkById(ctx context.Context, chunkId, documentId, taskId int64) (*entity.DocumentChunk, error)

	// SelectChunkListByParentBlockId 根据父块ID查询块列表
	SelectChunkListByParentBlockId(ctx context.Context, documentId, taskId, parentBlockId int64) ([]*entity.DocumentChunk, error)

	// ========== 父块相关 ==========

	// InsertParentBlock 插入父块
	InsertParentBlock(ctx context.Context, block *entity.DocumentParentBlock) error

	// InsertParentBlockBatch 批量插入父块
	InsertParentBlockBatch(ctx context.Context, blocks []*entity.DocumentParentBlock) error

	// DeleteParentBlockByDocumentId 根据文档ID删除父块
	DeleteParentBlockByDocumentId(ctx context.Context, documentId int64) error

	// SelectParentBlockListByIds 根据ID列表查询父块
	SelectParentBlockListByIds(ctx context.Context, ids []int64) ([]*entity.DocumentParentBlock, error)

	// SelectParentBlockById 根据ID查询父块
	SelectParentBlockById(ctx context.Context, blockId, documentId, taskId int64) (*entity.DocumentParentBlock, error)

	// ========== 结构节点相关 ==========

	// InsertStructureNodeBatch 批量插入结构节点
	InsertStructureNodeBatch(ctx context.Context, nodes []*entity.DocumentStructureNode) error

	// DeleteStructureNodeByDocumentId 根据文档ID删除结构节点
	DeleteStructureNodeByDocumentId(ctx context.Context, documentId int64) error

	// DeleteStructureNodeBatch 批量删除结构节点
	DeleteStructureNodeBatch(ctx context.Context, documentId int64, nodeIds []int64) error

	// SelectStructureNodeListByDocumentId 根据文档ID查询结构节点列表
	SelectStructureNodeListByDocumentId(ctx context.Context, documentId int64) ([]*entity.DocumentStructureNode, error)

	// ========== 属性相关 ==========

	// InsertProfile 插入文档属性
	InsertProfile(ctx context.Context, profile *entity.DocumentProfile) error

	// SelectProfileByDocumentId 根据文档ID查询文档属性
	SelectProfileByDocumentId(ctx context.Context, documentId int64) (*entity.DocumentProfile, error)

	// SaveProfile 创建或更新文档属性
	SaveProfile(ctx context.Context, profile *entity.DocumentProfile) error

	// DeleteProfileByDocumentId 根据文档ID删除属性
	DeleteProfileByDocumentId(ctx context.Context, documentId int64) error

	// SelectDocumentProfiles 查询所有文档属性
	SelectDocumentProfiles(ctx context.Context) ([]*entity.DocumentProfile, error)

	// ========== 话题关联相关 ==========

	// DeleteTopicDocumentRelationByDocumentId 根据文档ID删除话题关联
	DeleteTopicDocumentRelationByDocumentId(ctx context.Context, documentId int64) error
}
