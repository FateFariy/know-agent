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

	// SelectRetrievableDocumentsByIds 根据ID查询可检索的文档
	SelectRetrievableDocumentsByIds(ctx context.Context, documentIds ...int64) ([]*vo.DocumentMetadata, error)

	// SelectRetrievableDocumentsByKbIds 根据知识库ID查询可检索的文档
	SelectRetrievableDocumentsByKbIds(ctx context.Context, kbIds ...int64) ([]*vo.DocumentMetadata, error)

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

	// CountTaskByParams 统计任务数
	CountTaskByParams(ctx context.Context, documentId int64, taskType int, taskStatus []int) (int64, error)

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
	SelectStepListByPlanId(ctx context.Context, planId int64) (entity.DocumentStrategySteps, error)

	// UpdateStepExecuteStatus 更新步骤执行状态
	UpdateStepExecuteStatus(ctx context.Context, planId int64, status int) error

	// ========== 块相关 ==========

	// InsertChunk 插入块
	InsertChunk(ctx context.Context, chunk *entity.DocumentChunk) error

	// InsertChunkBatch 批量插入块
	InsertChunkBatch(ctx context.Context, chunks []*entity.DocumentChunk) error

	// UpdateChunkByTaskId 根据任务ID更新块
	UpdateChunkByTaskId(ctx context.Context, chunk *entity.DocumentChunk) error

	// UpdateBatchChunkById 根据ID批量更新块
	UpdateBatchChunkById(ctx context.Context, chunks []*entity.DocumentChunk, fields ...string) error

	// DeleteChunkByDocumentId 根据文档ID删除块
	DeleteChunkByDocumentId(ctx context.Context, documentId int64) error

	// SelectChunkPage 分页查询块
	SelectChunkPage(ctx context.Context, documentId, taskId int64, pageNo, pageSize int) ([]*entity.DocumentChunk, int64, error)

	// SelectChunkById 根据ID查询块
	SelectChunkById(ctx context.Context, chunkId, documentId, taskId int64) (*entity.DocumentChunk, error)

	// SelectChunkListByParentChunkId 根据父块ID查询块列表
	SelectChunkListByParentChunkId(ctx context.Context, documentId, taskId, parentChunkId int64) ([]*entity.DocumentChunk, error)

	// SelectChunks 根据条件查询块列表
	SelectChunks(ctx context.Context, where map[string]any) ([]*entity.DocumentChunk, error)

	// ========== 父块相关 ==========

	// InsertParentChunk 插入父块
	InsertParentChunk(ctx context.Context, chunk *entity.DocumentParentChunk) error

	// InsertParentChunkBatch 批量插入父块
	InsertParentChunkBatch(ctx context.Context, chunks []*entity.DocumentParentChunk) error

	// DeleteParentChunkByDocumentId 根据文档ID删除父块
	DeleteParentChunkByDocumentId(ctx context.Context, documentId int64) error

	// SelectParentChunkListByIds 根据ID列表查询父块
	SelectParentChunkListByIds(ctx context.Context, ids []int64) ([]*entity.DocumentParentChunk, error)

	// SelectParentChunkById 根据ID查询父块
	SelectParentChunkById(ctx context.Context, blockId, documentId, taskId int64) (*entity.DocumentParentChunk, error)

	// ========== 结构节点相关 ==========

	// InsertStructureNodeBatch 批量插入结构节点
	InsertStructureNodeBatch(ctx context.Context, nodes []*entity.StructureNode) error

	// DeleteStructureNodeByDocumentId 根据文档ID删除结构节点
	DeleteStructureNodeByDocumentId(ctx context.Context, documentId int64) error

	// SelectStructureNodeListByDocumentId 根据文档ID查询结构节点列表
	SelectStructureNodeListByDocumentId(ctx context.Context, documentId int64) ([]*entity.StructureNode, error)

	// SelectStructureNodeListByTask 根据文档ID和任务ID查询结构节点列表
	SelectStructureNodeListByTask(ctx context.Context, documentId, taskId int64) ([]*entity.StructureNode, error)

	// CountStructureNodes 统计结构节点数量
	CountStructureNodes(ctx context.Context, documentId, taskId int64) (int64, error)

	// ========== 属性相关 ==========

	// InsertProfile 插入文档属性
	InsertProfile(ctx context.Context, profile *entity.DocumentProfile) error

	// SelectProfileByDocumentId 根据文档ID查询文档属性
	SelectProfileByDocumentId(ctx context.Context, documentId int64) (*entity.DocumentProfile, error)

	// SaveProfile 创建或更新文档属性
	SaveProfile(ctx context.Context, profile *entity.DocumentProfile) error

	// DeleteProfileByDocumentId 根据文档ID删除属性
	DeleteProfileByDocumentId(ctx context.Context, documentId int64) error

	// SelectDocumentProfilesByDocIds 根据文档ID列表查询文档属性
	SelectDocumentProfilesByDocIds(ctx context.Context, documentIds []int64) ([]*entity.DocumentProfile, error)

	// SelectDocumentProfiles 查询所有文档属性
	SelectDocumentProfiles(ctx context.Context) ([]*entity.DocumentProfile, error)

	// ========== 话题关联相关 ==========

	// DeleteTopicDocumentRelationByDocumentId 根据文档ID删除话题关联
	DeleteTopicDocumentRelationByDocumentId(ctx context.Context, documentId int64) error

	// ========== 文档块相关 ==========

	// InsertDocumentBlockBatch 批量插入文档块
	InsertDocumentBlockBatch(ctx context.Context, blocks []*entity.DocumentBlock) error

	// SelectDocumentBlocksByTask 根据文档ID和任务ID查询文档块列表
	SelectDocumentBlocksByTask(ctx context.Context, documentId, taskId int64) ([]*entity.DocumentBlock, error)

	// DeleteDocumentBlocksByTask 根据文档ID和任务ID删除文档块
	DeleteDocumentBlocksByTask(ctx context.Context, documentId, taskId int64) error

	// DeleteDocumentBlocksByDocumentId 根据文档ID删除文档块
	DeleteDocumentBlocksByDocumentId(ctx context.Context, documentId int64) error

	// CountDocumentBlocks 统计文档块数量
	CountDocumentBlocks(ctx context.Context, documentId, taskId int64) (int64, error)

	// SelectDocumentBlocksWithLimit 查询文档块列表（带限制）
	SelectDocumentBlocksWithLimit(ctx context.Context, documentId, taskId int64, limit int) ([]*entity.DocumentBlock, error)

	// SelectDocumentBlockPageNumbers 查询文档块中不重复的页码列表
	SelectDocumentBlockPageNumbers(ctx context.Context, documentId, taskId int64) ([]int, error)

	// ========== 知识库统计相关 ==========

	// CountDocumentsByKbIds 按知识库ID列表统计文档数量（返回 map[knowledgeBaseId]count）
	CountDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error)

	// CountRetrievableDocumentsByKbIds 按知识库ID列表统计可检索文档数量（返回 map[knowledgeBaseId]count）
	CountRetrievableDocumentsByKbIds(ctx context.Context, kbIds []int64) (map[int64]int64, error)
}

type TableRepository interface {
	// InsertTable 插入表格
	InsertTable(ctx context.Context, table *entity.DocumentTable) error

	// InsertTableColumnBatch 批量插入表格列
	InsertTableColumnBatch(ctx context.Context, columns []*entity.TableColumn) error

	// InsertTableRowBatch 批量插入表格行
	InsertTableRowBatch(ctx context.Context, rows []*entity.TableRow) error

	// InsertTableCellBatch 批量插入表格单元格
	InsertTableCellBatch(ctx context.Context, cells []*entity.TableCell) error

	// SelectTableById 根据ID查询表格
	SelectTableById(ctx context.Context, tableId int64) (*entity.DocumentTable, error)

	// SelectTablesByTask 根据文档ID和任务ID查询表格列表
	SelectTablesByTask(ctx context.Context, documentId, taskId int64) ([]*entity.DocumentTable, error)

	// SelectTableColumnsByTableId 根据表格ID查询列列表
	SelectTableColumnsByTableId(ctx context.Context, tableId int64) ([]*entity.TableColumn, error)

	// SelectTableRowsByTableId 根据表格ID查询行列表
	SelectTableRowsByTableId(ctx context.Context, tableId int64) ([]*entity.TableRow, error)

	// SelectTableCellsByTableId 根据表格ID查询单元格列表
	SelectTableCellsByTableId(ctx context.Context, tableId int64) ([]*entity.TableCell, error)

	// DeleteTableDetailByTableIds 根据表格ID列表删除表格列、行、单元格
	DeleteTableDetailByTableIds(ctx context.Context, tableIds []int64) error

	// DeleteTablesByTask 根据文档ID和任务ID删除表格
	DeleteTablesByTask(ctx context.Context, documentId, taskId int64) error

	// DeleteTablesByDocumentId 根据文档ID删除表格
	DeleteTablesByDocumentId(ctx context.Context, documentId int64) error

	// CountTables 统计表格数量
	CountTables(ctx context.Context, documentId, taskId int64) (int64, error)

	// SelectTablePageNumbers 查询表格中不重复的页码列表
	SelectTablePageNumbers(ctx context.Context, documentId, taskId int64) ([]int, error)
}
