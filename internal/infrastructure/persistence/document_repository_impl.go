package persistence

import (
	"context"
	"errors"
	"slices"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/infrastructure/model"
	"github.com/swiftbit/know-agent/internal/svc"
)

const (
	ticketUserList      = "ticket_user_list:%d"
	userLogin           = "user_login:%d:%s"
	loginMobileErrorKey = "login:mobile:error:%s"
	loginEmailErrorKey  = "login:email:error:%s"
)

type DocumentRepositoryImpl struct {
	db  *gorm.DB
	rdb *redis.Client
}

var _ adapter.DocumentRepository = (*DocumentRepositoryImpl)(nil)

func NewDocumentRepository(svcCtx *svc.ServiceContext) *DocumentRepositoryImpl {
	return &DocumentRepositoryImpl{
		db:  svcCtx.Db,
		rdb: svcCtx.Rdb,
	}
}

// ========== 文档聚合根相关 ==========

// InsertDocumentAggregate 插入文档聚合根
func (d *DocumentRepositoryImpl) InsertDocumentAggregate(ctx context.Context, agg *aggregate.Document) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(convert.ToDocumentModel(agg.Document)).Error; err != nil {
			return err
		}
		if err := tx.Create(convert.ToDocumentTaskModel(agg.Task)).Error; err != nil {
			return err
		}
		if err := tx.Create(convert.ToDocumentTaskLogModel(agg.TaskLog)).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DocumentRepositoryImpl) UpdateDocumentAggregate(ctx context.Context, aggregate *aggregate.Document) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Updates(convert.ToDocumentModel(aggregate.Document)).Error; err != nil {
			return err
		}
		if err := tx.Updates(convert.ToDocumentTaskModel(aggregate.Task)).Error; err != nil {
			return err
		}
		if err := tx.Updates(convert.ToDocumentTaskLogModel(aggregate.TaskLog)).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DocumentRepositoryImpl) InsertOrUpdateDocumentAggregate(ctx context.Context, agg *aggregate.Document) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Updates(convert.ToDocumentModel(agg.Document)).Error; err != nil {
			return err
		}
		if err := tx.Create(convert.ToDocumentTaskModel(agg.Task)).Error; err != nil {
			return err
		}
		if err := tx.Create(convert.ToDocumentTaskLogModel(agg.TaskLog)).Error; err != nil {
			return err
		}
		return nil
	})
}

// ========== 文档相关 ==========

// SelectDocumentPage 获取文档分页列表
func (d *DocumentRepositoryImpl) SelectDocumentPage(ctx context.Context, pageNo, pageSize int, keyword string) ([]*entity.Document, int64, error) {
	var documents []*entity.Document
	query := d.db.WithContext(ctx).Model(&model.Document{}).Scopes(utils.Paginate(pageNo, pageSize))
	if keyword != "" {
		query = query.Where("document_name LIKE %?% or original_file_name LIKE %?%", keyword, keyword)
	}
	res := query.Order("update_time DESC").Find(&documents)
	return documents, res.RowsAffected, res.Error
}

// SelectDocumentById 获取文档
func (d *DocumentRepositoryImpl) SelectDocumentById(ctx context.Context, documentId int64) (*entity.Document, error) {
	var document = &entity.Document{ID: documentId}
	if err := d.db.WithContext(ctx).Model(&model.Document{}).First(document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrDocumentNotFound.Format(documentId)
		}
		return nil, err
	}
	return document, nil
}

func (d *DocumentRepositoryImpl) UpdateDocument(ctx context.Context, document *entity.Document) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteDocumentById(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

// ========== 任务相关 ==========

// InsertTask 插入任务
func (d *DocumentRepositoryImpl) InsertTask(ctx context.Context, task *entity.DocumentTask) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) UpdateTask(ctx context.Context, task *entity.DocumentTask) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteTaskByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectTaskById(ctx context.Context, taskId int64) (*entity.DocumentTask, error) {
	var task = &entity.DocumentTask{ID: taskId}
	if err := d.db.WithContext(ctx).Model(&model.DocumentTask{}).First(task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrTaskNotFound.Format(taskId)
		}
		return nil, err
	}
	return task, nil
}

// SelectLatestTask 获取最新任务
func (d *DocumentRepositoryImpl) SelectLatestTask(ctx context.Context, documentId int64) (*entity.DocumentTask, error) {
	var task *entity.DocumentTask
	res := d.db.WithContext(ctx).Model(&model.DocumentTask{}).Where("document_id = ?", documentId).Order("id DESC").First(task)
	return task, res.Error
}

// SelectTaskListByDocumentIds 获取任务列表
func (d *DocumentRepositoryImpl) SelectTaskListByDocumentIds(ctx context.Context, documentIds []int64) ([]*entity.DocumentTask, error) {
	var tasks []*entity.DocumentTask
	res := d.db.WithContext(ctx).Model(&model.DocumentTask{}).Where("document_id IN ?", documentIds).Order("id DESC").Find(&tasks)
	return tasks, res.Error
}

// CountActiveTask 统计活跃任务数量
func (d *DocumentRepositoryImpl) CountActiveTask(ctx context.Context, documentId int64, taskType int, taskStatus ...int) (int64, error) {
	var count int64
	var err error
	query := d.db.WithContext(ctx).Model(&model.DocumentTask{}).Where("document_id = ?", documentId)
	if taskType != 0 {
		query.Where("task_type = ?", taskType)
	}
	if len(taskStatus) > 0 {
		err = query.Where("task_status IN ?", taskStatus).Count(&count).Error
	}
	return count, err
}

// ========== 任务日志相关 ==========

func (d *DocumentRepositoryImpl) InsertTaskLog(ctx context.Context, log *entity.DocumentTaskLog) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteTaskLogByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectTaskLogPage(ctx context.Context, taskId int64, pageNo, pageSize int) ([]*entity.DocumentTaskLog, int64, error) {
	// TODO implement me
	panic("implement me")
}

// ========== 方案/策略相关 ==========

func (d *DocumentRepositoryImpl) InsertPlan(ctx context.Context, plan *entity.DocumentStrategyPlan) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) UpdatePlan(ctx context.Context, plan *entity.DocumentStrategyPlan) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeletePlanByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectPlanById(ctx context.Context, planId int64) (*entity.DocumentStrategyPlan, error) {
	var plan = &entity.DocumentStrategyPlan{ID: planId}
	if err := d.db.WithContext(ctx).Model(&model.DocumentStrategyPlan{}).First(plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrStrategyPlanNotFound.Format(planId)
		}
		return nil, err
	}
	return plan, nil
}

func (d *DocumentRepositoryImpl) SelectLatestPlanVersion(ctx context.Context, documentId int64) (int, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) UpdatePlanStatus(ctx context.Context, planId int64, status int) error {
	// TODO implement me
	panic("implement me")
}

// ========== 步骤相关 ==========

func (d *DocumentRepositoryImpl) InsertStep(ctx context.Context, step *entity.DocumentStrategyStep) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteStepByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectStepListByPlanId(ctx context.Context, planId int64) ([]*entity.DocumentStrategyStep, error) {
	var steps []*entity.DocumentStrategyStep
	res := d.db.WithContext(ctx).Model(&model.DocumentStrategyStep{}).Where("plan_id = ?", planId).Find(&steps)
	slices.SortFunc(steps, func(a, b *entity.DocumentStrategyStep) int {
		if a.PipelineType != b.PipelineType {
			return a.PipelineType - b.PipelineType
		} else if a.StepNo != b.StepNo {
			return a.StepNo - b.StepNo
		}
		return int(a.ID - b.ID)
	})
	return steps, res.Error
}

func (d *DocumentRepositoryImpl) UpdateStepExecuteStatus(ctx context.Context, planId int64, status int) error {
	// TODO implement me
	panic("implement me")
}

// ========== 块相关 ==========

func (d *DocumentRepositoryImpl) InsertChunk(ctx context.Context, chunk *entity.DocumentChunk) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) InsertChunkBatch(ctx context.Context, chunks []*entity.DocumentChunk) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) UpdateChunk(ctx context.Context, chunk *entity.DocumentChunk) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteChunkByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectChunkPage(ctx context.Context, documentId, taskId int64, pageNo, pageSize int) ([]*entity.DocumentChunk, int64, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectChunkById(ctx context.Context, chunkId, documentId, taskId int64) (*entity.DocumentChunk, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectChunkListByParentBlockId(ctx context.Context, documentId, taskId, parentBlockId int64) ([]*entity.DocumentChunk, error) {
	// TODO implement me
	panic("implement me")
}

// ========== 父块相关 ==========

func (d *DocumentRepositoryImpl) InsertParentBlock(ctx context.Context, block *entity.DocumentParentBlock) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) InsertParentBlockBatch(ctx context.Context, blocks []*entity.DocumentParentBlock) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteParentBlockByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectParentBlockListByIds(ctx context.Context, ids []int64) ([]*entity.DocumentParentBlock, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectParentBlockById(ctx context.Context, blockId, documentId, taskId int64) (*entity.DocumentParentBlock, error) {
	// TODO implement me
	panic("implement me")
}

// ========== 结构节点相关 ==========

func (d *DocumentRepositoryImpl) InsertStructureNode(ctx context.Context, node *entity.DocumentStructureNode) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) InsertStructureNodeBatch(ctx context.Context, nodes []*entity.DocumentStructureNode) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteStructureNodeByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteStructureNodeBatch(ctx context.Context, documentId int64, nodeIds []int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectStructureNodeListByDocumentId(ctx context.Context, documentId int64) ([]*entity.DocumentStructureNode, error) {
	// TODO implement me
	panic("implement me")
}

// ========== 属性相关 ==========

func (d *DocumentRepositoryImpl) InsertProfile(ctx context.Context, profile *entity.DocumentProfile) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteProfileByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

// ========== 话题关联相关 ==========

func (d *DocumentRepositoryImpl) DeleteTopicDocumentRelationByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}
