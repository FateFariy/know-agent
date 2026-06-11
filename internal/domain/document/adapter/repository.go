package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// DocumentRepository 文档数据访问接口
type DocumentRepository interface {
	// InsertDocumentAggregate 插入文档聚合根
	InsertDocumentAggregate(ctx context.Context, aggregate *aggregate.Document) error

	// UpdateDocument 更新文档
	UpdateDocument(ctx context.Context, document *entity.Document) error

	// DeleteDocumentById 删除文档
	DeleteDocumentById(ctx context.Context, documentId int64) error

	// SelectDocumentById 根据ID查询文档
	SelectDocumentById(ctx context.Context, documentId int64) (*entity.Document, error)

	// SelectDocumentPage 分页查询文档
	SelectDocumentPage(ctx context.Context, pageNo, pageSize int, keyword string) ([]*entity.Document, int64, error)

	// InsertTask 插入任务
	InsertTask(ctx context.Context, task *entity.DocumentTask) error

	// UpdateTask 更新任务
	UpdateTask(ctx context.Context, task *entity.DocumentTask) error

	// DeleteTaskByDocumentId 根据文档ID删除任务
	DeleteTaskByDocumentId(ctx context.Context, documentId int64) error

	// SelectTaskById 根据ID查询任务
	SelectTaskById(ctx context.Context, taskId int64) (*entity.DocumentTask, error)

	// SelectLatestTask 查询最新任务
	SelectLatestTask(ctx context.Context, documentId int64, taskType ...int) (*entity.DocumentTask, error)

	// CountActiveTask 统计活跃任务数
	CountActiveTask(ctx context.Context, documentId int64, taskType ...int) (int64, error)

	// SelectTaskListByDocumentIds 根据文档ID列表查询任务
	SelectTaskListByDocumentIds(ctx context.Context, documentIds []int64) ([]*entity.DocumentTask, error)

	// InsertTaskLog 插入任务日志
	InsertTaskLog(ctx context.Context, log *entity.DocumentTaskLog) error

	// DeleteTaskLogByDocumentId 根据文档ID删除任务日志
	DeleteTaskLogByDocumentId(ctx context.Context, documentId int64) error

	// SelectTaskLogPage 分页查询任务日志
	SelectTaskLogPage(ctx context.Context, taskId int64, pageNo, pageSize int) ([]*entity.DocumentTaskLog, int64, error)

	// InsertPlan 插入方案
	InsertPlan(ctx context.Context, plan *entity.DocumentStrategyPlan) error

	// UpdatePlan 更新方案
	UpdatePlan(ctx context.Context, plan *entity.DocumentStrategyPlan) error

	// DeletePlanByDocumentId 根据文档ID删除方案
	DeletePlanByDocumentId(ctx context.Context, documentId int64) error

	// SelectPlanById 根据ID查询方案
	SelectPlanById(ctx context.Context, planId int64) (*entity.DocumentStrategyPlan, error)

	// SelectLatestPlanVersion 查询最新方案版本
	SelectLatestPlanVersion(ctx context.Context, documentId int64) (int, error)

	// InsertStep 插入步骤
	InsertStep(ctx context.Context, step *entity.DocumentStrategyStep) error

	// DeleteStepByDocumentId 根据文档ID删除步骤
	DeleteStepByDocumentId(ctx context.Context, documentId int64) error

	// SelectStepListByPlanId 根据方案ID查询步骤列表
	SelectStepListByPlanId(ctx context.Context, planId int64) ([]*entity.DocumentStrategyStep, error)

	// DeleteChunkByDocumentId 根据文档ID删除块
	DeleteChunkByDocumentId(ctx context.Context, documentId int64) error

	// SelectChunkPage 分页查询块
	SelectChunkPage(ctx context.Context, documentId, taskId int64, pageNo, pageSize int) ([]*entity.DocumentChunk, int64, error)

	// SelectChunkById 根据ID查询块
	SelectChunkById(ctx context.Context, chunkId, documentId, taskId int64) (*entity.DocumentChunk, error)

	// SelectChunkListByParentBlockId 根据父块ID查询块列表
	SelectChunkListByParentBlockId(ctx context.Context, documentId, taskId, parentBlockId int64) ([]*entity.DocumentChunk, error)

	// DeleteParentBlockByDocumentId 根据文档ID删除父块
	DeleteParentBlockByDocumentId(ctx context.Context, documentId int64) error

	// SelectParentBlockListByIds 根据ID列表查询父块
	SelectParentBlockListByIds(ctx context.Context, ids []int64) ([]*entity.DocumentParentBlock, error)

	// SelectParentBlockById 根据ID查询父块
	SelectParentBlockById(ctx context.Context, blockId, documentId, taskId int64) (*entity.DocumentParentBlock, error)

	// DeleteProfileByDocumentId 根据文档ID删除属性
	DeleteProfileByDocumentId(ctx context.Context, documentId int64) error

	// DeleteTopicDocumentRelationByDocumentId 根据文档ID删除话题关联
	DeleteTopicDocumentRelationByDocumentId(ctx context.Context, documentId int64) error
}
