package persistence

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	errorx "github.com/swiftbit/know-agent/internal/error"
)

const (
	ticketUserList      = "ticket_user_list:%d"
	userLogin           = "user_login:%d:%s"
	loginMobileErrorKey = "login:mobile:error:%s"
	loginEmailErrorKey  = "login:email:error:%s"
)

type DocumentRepositoryImpl struct {
	db    *gorm.DB
	rdb   *redis.Client
	cache cache.Cache
}

var _ adapter.DocumentRepository = (*DocumentRepositoryImpl)(nil)

func NewDocumentRepository(db *gorm.DB, rdb *redis.Client, cache cache.Cache) *DocumentRepositoryImpl {
	return &DocumentRepositoryImpl{
		db:    db,
		rdb:   rdb,
		cache: cache,
	}
}

// InsertDocumentAggregate 插入文档聚合根
func (d *DocumentRepositoryImpl) InsertDocumentAggregate(ctx context.Context, aggregate *aggregate.Document) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(convert.ToDocumentModel(aggregate.Document)).Error; err != nil {
			return err
		}
		if err := tx.Create(convert.ToDocumentTaskModel(aggregate.Task)).Error; err != nil {
			return err
		}
		if err := tx.Create(convert.ToDocumentTaskLogModel(aggregate.TaskLog)).Error; err != nil {
			return err
		}
		return nil
	})
}

func (d *DocumentRepositoryImpl) UpdateDocument(ctx context.Context, document *entity.Document) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteDocumentById(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectDocumentById(ctx context.Context, documentId int64) (*entity.Document, error) {
	var document = entity.Document{ID: documentId}
	if err := d.db.WithContext(ctx).Model(&entity.Document{}).First(&document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrDocumentNotFound
		}
		return nil, err
	}
	return &document, nil
}

// SelectDocumentPage 获取文档分页列表
func (d *DocumentRepositoryImpl) SelectDocumentPage(ctx context.Context, pageNo, pageSize int, keyword string) ([]*entity.Document, int64, error) {
	var documents []*entity.Document
	query := d.db.WithContext(ctx).Model(&entity.Document{}).Scopes(utils.Paginate(pageNo, pageSize))
	if keyword != "" {
		query = query.Where("document_name LIKE %?% or original_file_name LIKE %?%", keyword, keyword)
	}
	res := query.Order("update_time DESC").Find(&documents)
	return documents, res.RowsAffected, res.Error
}

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
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectLatestTask(ctx context.Context, documentId int64, taskType ...int) (*entity.DocumentTask, error) {
	// TODO implement me
	panic("implement me")
}

// CountActiveTask 统计活跃任务数量
func (d *DocumentRepositoryImpl) CountActiveTask(ctx context.Context, documentId int64, taskStatus ...int) (int64, error) {
	var count int64
	var err error
	query := d.db.WithContext(ctx).Model(&entity.DocumentTask{}).Where("document_id = ?", documentId)
	if len(taskStatus) > 0 {
		err = query.Where("status IN ?", taskStatus).Count(&count).Error
	}
	return count, err
}

func (d *DocumentRepositoryImpl) SelectTaskListByDocumentIds(ctx context.Context, documentIds []int64) ([]*entity.DocumentTask, error) {
	var tasks []*entity.DocumentTask
	res := d.db.WithContext(ctx).Model(&entity.DocumentTask{}).Where("document_id IN ?", documentIds).Order("id DESC").Find(&tasks)
	return tasks, res.Error
}

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
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectLatestPlanVersion(ctx context.Context, documentId int64) (int, error) {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) InsertStep(ctx context.Context, step *entity.DocumentStrategyStep) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteStepByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectStepListByPlanId(ctx context.Context, planId int64) ([]*entity.DocumentStrategyStep, error) {
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

func (d *DocumentRepositoryImpl) DeleteProfileByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) DeleteTopicDocumentRelationByDocumentId(ctx context.Context, documentId int64) error {
	// TODO implement me
	panic("implement me")
}
