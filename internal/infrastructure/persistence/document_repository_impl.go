package persistence

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/infrastructure/model"
	"github.com/swiftbit/know-agent/internal/svc"
)

type DocumentRepositoryImpl struct {
	rdb     *redis.Client
	vdb     adapter.VectorDB
	storage adapter.Storage
	*transactionManager
}

var _ adapter.DocumentRepository = (*DocumentRepositoryImpl)(nil)

func NewDocumentRepository(svcCtx *svc.ServiceContext, storage adapter.Storage, vdb adapter.VectorDB) *DocumentRepositoryImpl {
	return &DocumentRepositoryImpl{
		transactionManager: &transactionManager{db: svcCtx.Db},
		rdb:                svcCtx.Rdb,
		storage:            storage,
		vdb:                vdb,
	}
}

// DeleteDocumentRelatedDataById 删除文档关联数据
func (d *DocumentRepositoryImpl) DeleteDocumentRelatedDataById(ctx context.Context, documentId int64) (string, error) {
	var documentName string
	err := d.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		document, err := d.SelectDocumentById(ctx, documentId)
		if err != nil {
			return err
		}
		documentName = document.DocumentName

		// 删除存储对象
		if err = d.storage.DeleteObjects(ctx, []string{document.ObjectName, document.ParseTextPath}); err != nil {
			return err
		}

		// 删除向量索引
		if err = d.vdb.DeleteVectorByDocumentId(ctx, documentId); err != nil {
			return err
		}

		// 删除其他索引（TODO: 实现关键词搜索、导航索引、知识路由索引、结构图投影）

		// 删除相关数据
		if err = d.DeleteProfileByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteTopicDocumentRelationByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteParentBlockByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteChunkByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteStructureNodeByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteTaskLogByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteStepByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteTaskByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeletePlanByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteDocumentById(ctx, documentId); err != nil {
			return err
		}
		return nil
	})
	return documentName, err
}

// ========== 文档相关 ==========

// InsertDocument 插入文档
func (d *DocumentRepositoryImpl) InsertDocument(ctx context.Context, document *entity.Document) error {
	return d.dbWithContext(ctx).Create(convert.ToDocumentModel(document)).Error
}

// SelectDocumentPage 获取文档分页列表
func (d *DocumentRepositoryImpl) SelectDocumentPage(ctx context.Context, pageNo, pageSize int, keyword string) ([]*entity.Document, int64, error) {
	var documents []*entity.Document
	query := d.dbWithContext(ctx).Model(&model.Document{}).Scopes(utils.Paginate(pageNo, pageSize))
	if keyword != "" {
		query = query.Where("document_name LIKE %?% or original_file_name LIKE %?%", keyword, keyword)
	}
	res := query.Order("update_time DESC").Find(&documents)
	return documents, res.RowsAffected, res.Error
}

// SelectDocumentById 获取文档
func (d *DocumentRepositoryImpl) SelectDocumentById(ctx context.Context, documentId int64) (*entity.Document, error) {
	document := &entity.Document{ID: documentId}
	if err := d.dbWithContext(ctx).Model(&model.Document{}).First(document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrDocumentNotFound.Format(documentId)
		}
		return nil, err
	}
	return document, nil
}

// UpdateDocumentById 根据ID更新文档
func (d *DocumentRepositoryImpl) UpdateDocumentById(ctx context.Context, document *entity.Document) error {
	return d.dbWithContext(ctx).Where("id = ?", document.ID).Updates(convert.ToDocumentModel(document)).Error
}

// DeleteDocumentById  根据ID删除文档
func (d *DocumentRepositoryImpl) DeleteDocumentById(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("id = ?", documentId).Delete(&model.Document{}).Error
}

// SelectRetrievableDocuments 查询可检索的文档
func (d *DocumentRepositoryImpl) SelectRetrievableDocuments(ctx context.Context, documentIds ...int64) ([]*vo.KnowledgeDocument, error) {
	var documents []*vo.KnowledgeDocument
	query := d.dbWithContext(ctx).Model(&model.Document{}).
		Where("index_status = ? AND last_index_task_id IS NOT NULL", vo.IndexStatusBuildSuccess)

	if len(documentIds) > 0 {
		query = query.Where("id IN ?", documentIds)
	}
	if err := query.Order("update_time DESC, id DESC").Find(&documents).Error; err != nil {
		return nil, err
	}
	return documents, nil
}

// ========== 任务相关 ==========

// InsertTask 插入任务
func (d *DocumentRepositoryImpl) InsertTask(ctx context.Context, task *entity.DocumentTask) error {
	return d.dbWithContext(ctx).Create(convert.ToDocumentTaskModel(task)).Error
}

// UpdateTaskById 根据任务ID更新任务
func (d *DocumentRepositoryImpl) UpdateTaskById(ctx context.Context, task *entity.DocumentTask) error {
	return d.dbWithContext(ctx).Where("id = ?", task.ID).Updates(convert.ToDocumentTaskModel(task)).Error
}

// DeleteTaskByDocumentId  根据文档ID删除任务
func (d *DocumentRepositoryImpl) DeleteTaskByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentTask{}).Error
}

// SelectTaskById 根据任务ID获取任务
func (d *DocumentRepositoryImpl) SelectTaskById(ctx context.Context, taskId int64) (*entity.DocumentTask, error) {
	task := &entity.DocumentTask{ID: taskId}
	if err := d.dbWithContext(ctx).Model(&model.DocumentTask{}).First(task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrTaskNotFound.Format(taskId)
		}
		return nil, err
	}
	return task, nil
}

// SelectLatestTask 根据文档ID获取最新任务
func (d *DocumentRepositoryImpl) SelectLatestTask(ctx context.Context, documentId int64, taskTypes ...int) (*entity.DocumentTask, error) {
	task := &entity.DocumentTask{}
	query := d.dbWithContext(ctx).Model(&model.DocumentTask{}).Where("document_id = ?", documentId).Order("id DESC")
	if len(taskTypes) > 0 {
		query = query.Where("task_type IN ?", taskTypes)
	}
	if err := query.First(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

// SelectTaskListByDocumentIds 获取任务列表
func (d *DocumentRepositoryImpl) SelectTaskListByDocumentIds(ctx context.Context, documentIds []int64) ([]*entity.DocumentTask, error) {
	var tasks []*entity.DocumentTask
	res := d.dbWithContext(ctx).Model(&model.DocumentTask{}).Where("document_id IN ?", documentIds).Order("id DESC").Find(&tasks)
	return tasks, res.Error
}

// CountActiveTask 统计活跃任务数量
func (d *DocumentRepositoryImpl) CountActiveTask(ctx context.Context, documentId int64, taskType int, taskStatus ...int) (int64, error) {
	var count int64
	var err error
	query := d.dbWithContext(ctx).Model(&model.DocumentTask{}).Where("document_id = ?", documentId)
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
	return d.dbWithContext(ctx).Create(convert.ToDocumentTaskLogModel(log)).Error
}

// DeleteTaskLogByDocumentId  根据文档ID删除任务日志
func (d *DocumentRepositoryImpl) DeleteTaskLogByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentTaskLog{}).Error
}

// SelectTaskLogPage 根据任务ID查询任务日志分页列表
func (d *DocumentRepositoryImpl) SelectTaskLogPage(ctx context.Context, taskId int64, pageNo, pageSize int) ([]*entity.DocumentTaskLog, int64, error) {
	var logs []*entity.DocumentTaskLog
	var total int64
	query := d.dbWithContext(ctx).Model(&model.DocumentTaskLog{}).Where("task_id = ?", taskId)
	if err := query.Scopes(utils.Paginate(pageNo, pageSize)).Order("create_time ASC, id ASC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// ========== 方案/策略相关 ==========

// InsertPlan 插入方案/策略
func (d *DocumentRepositoryImpl) InsertPlan(ctx context.Context, plan *entity.DocumentStrategyPlan) error {
	return d.dbWithContext(ctx).Create(convert.ToDocumentStrategyPlanModel(plan)).Error
}

// UpdatePlanById 根据方案/策略ID更新方案/策略
func (d *DocumentRepositoryImpl) UpdatePlanById(ctx context.Context, plan *entity.DocumentStrategyPlan) error {
	return d.dbWithContext(ctx).Where("id = ?", plan.ID).Updates(convert.ToDocumentStrategyPlanModel(plan)).Error
}

// DeletePlanByDocumentId  根据文档ID删除方案/策略
func (d *DocumentRepositoryImpl) DeletePlanByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentStrategyPlan{}).Error
}

// SelectPlanById 根据方案/策略ID获取方案/策略
func (d *DocumentRepositoryImpl) SelectPlanById(ctx context.Context, planId int64) (*entity.DocumentStrategyPlan, error) {
	plan := &entity.DocumentStrategyPlan{ID: planId}
	if err := d.dbWithContext(ctx).Model(&model.DocumentStrategyPlan{}).First(plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrStrategyPlanNotFound.Format(planId)
		}
		return nil, err
	}
	return plan, nil
}

// SelectLatestPlanVersion 根据文档ID获取最新方案/策略版本
func (d *DocumentRepositoryImpl) SelectLatestPlanVersion(ctx context.Context, documentId int64) (int, error) {
	plan := &model.DocumentStrategyPlan{DocumentId: documentId}
	err := d.dbWithContext(ctx).Select("plan_version").Where(plan).Order("plan_version DESC").First(plan).Error
	return plan.PlanVersion, err
}

// ========== 步骤相关 ==========

func (d *DocumentRepositoryImpl) InsertStepBatch(ctx context.Context, steps []*entity.DocumentStrategyStep) error {
	return d.dbWithContext(ctx).CreateInBatches(convert.ToDocumentStrategyStepModelList(steps), 100).Error
}

// DeleteStepByDocumentId  根据文档ID删除步骤
func (d *DocumentRepositoryImpl) DeleteStepByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentStrategyStep{}).Error
}

// SelectStepListByPlanId  根据方案/策略ID查询步骤列表
func (d *DocumentRepositoryImpl) SelectStepListByPlanId(ctx context.Context, planId int64) ([]*entity.DocumentStrategyStep, error) {
	var steps []*entity.DocumentStrategyStep
	if err := d.dbWithContext(ctx).Model(&model.DocumentStrategyStep{}).Where("plan_id = ?", planId).Find(&steps).Error; err != nil {
		return nil, err
	}
	slices.SortFunc(steps, func(a, b *entity.DocumentStrategyStep) int {
		if a.PipelineType != b.PipelineType {
			return strings.Compare(a.PipelineType, b.PipelineType)
		} else if a.StepNo != b.StepNo {
			return a.StepNo - b.StepNo
		}
		return int(a.ID - b.ID)
	})
	return steps, nil
}

// UpdateStepExecuteStatus 根据方案/策略ID更新步骤执行状态
func (d *DocumentRepositoryImpl) UpdateStepExecuteStatus(ctx context.Context, planId int64, status int) error {
	return d.dbWithContext(ctx).Model(&model.DocumentStrategyStep{}).Where("plan_id = ?", planId).Update("execute_status", status).Error
}

// ========== 块相关 ==========

func (d *DocumentRepositoryImpl) InsertChunk(ctx context.Context, chunk *entity.DocumentChunk) error {
	return d.dbWithContext(ctx).Create(convert.ToDocumentChunkModel(chunk)).Error
}

func (d *DocumentRepositoryImpl) InsertChunkBatch(ctx context.Context, chunks []*entity.DocumentChunk) error {
	return d.dbWithContext(ctx).CreateInBatches(convert.ToDocumentChunkModelList(chunks), 100).Error
}

// UpdateChunkByTaskId 根据任务ID更新块
func (d *DocumentRepositoryImpl) UpdateChunkByTaskId(ctx context.Context, chunk *entity.DocumentChunk) error {
	return d.dbWithContext(ctx).Where("task_id = ?", chunk.TaskId).Updates(convert.ToDocumentChunkModel(chunk)).Error
}

// DeleteChunkByDocumentId 根据文档ID删除块
func (d *DocumentRepositoryImpl) DeleteChunkByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentChunk{}).Error
}

// SelectChunkPage 根据文档ID查询块分页列表
func (d *DocumentRepositoryImpl) SelectChunkPage(ctx context.Context, documentId, taskId int64, pageNo, pageSize int) ([]*entity.DocumentChunk, int64, error) {
	var chunks []*entity.DocumentChunk
	var total int64
	query := d.dbWithContext(ctx).Model(&model.DocumentChunk{}).Where("document_id = ? AND task_id = ?", documentId, taskId)
	if err := query.Scopes(utils.Paginate(pageNo, pageSize)).Order("id ASC").Find(&chunks).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	return chunks, total, nil
}

// SelectChunkById 根据块ID查询块详情
func (d *DocumentRepositoryImpl) SelectChunkById(ctx context.Context, chunkId, documentId, taskId int64) (*entity.DocumentChunk, error) {
	chunk := &entity.DocumentChunk{}
	if err := d.dbWithContext(ctx).Model(&model.DocumentChunk{}).
		Where("id = ? AND document_id = ? AND task_id = ?", chunkId, documentId, taskId).
		First(chunk).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NewBizError(errorx.ErrDocumentNotFound.Code, "chunk 详情不存在")
		}
		return nil, err
	}
	return chunk, nil
}

// SelectChunkListByParentBlockId 根据父块ID查询块列表
func (d *DocumentRepositoryImpl) SelectChunkListByParentBlockId(ctx context.Context, documentId, taskId, parentBlockId int64) ([]*entity.DocumentChunk, error) {
	var chunks []*entity.DocumentChunk
	if err := d.dbWithContext(ctx).Model(&model.DocumentChunk{}).
		Where("document_id = ? AND task_id = ? AND parent_block_id = ?", documentId, taskId, parentBlockId).
		Order("chunk_no ASC").Find(&chunks).Error; err != nil {
		return nil, err
	}
	return chunks, nil
}

// ========== 父块相关 ==========

func (d *DocumentRepositoryImpl) InsertParentBlock(ctx context.Context, block *entity.DocumentParentBlock) error {
	return d.dbWithContext(ctx).Create(convert.ToDocumentParentBlockModel(block)).Error
}

func (d *DocumentRepositoryImpl) InsertParentBlockBatch(ctx context.Context, blocks []*entity.DocumentParentBlock) error {
	return d.dbWithContext(ctx).CreateInBatches(convert.ToDocumentParentBlockModelList(blocks), 100).Error
}

// DeleteParentBlockByDocumentId 根据文档ID删除父块
func (d *DocumentRepositoryImpl) DeleteParentBlockByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentParentBlock{}).Error
}

// SelectParentBlockListByIds 根据父块ID列表查询父块列表
func (d *DocumentRepositoryImpl) SelectParentBlockListByIds(ctx context.Context, ids []int64) ([]*entity.DocumentParentBlock, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var parentBlocks []*entity.DocumentParentBlock
	if err := d.dbWithContext(ctx).Model(&model.DocumentParentBlock{}).
		Where("id IN ?", ids).
		Order("parent_no ASC").
		Find(&parentBlocks).Error; err != nil {
		return nil, err
	}
	return parentBlocks, nil
}

// SelectParentBlockById 根据父块ID查询父块详情
func (d *DocumentRepositoryImpl) SelectParentBlockById(ctx context.Context, blockId, documentId, taskId int64) (*entity.DocumentParentBlock, error) {
	parentBlock := &entity.DocumentParentBlock{}
	if err := d.dbWithContext(ctx).Model(&model.DocumentParentBlock{}).Where("id = ? AND document_id = ? AND task_id = ?", blockId, documentId, taskId).First(parentBlock).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NewBizError(errorx.ErrDocumentNotFound.Code, "父块详情不存在")
		}
		return nil, err
	}
	return parentBlock, nil
}

// ========== 结构节点相关 ==========

// InsertStructureNodeBatch 批量插入结构节点
func (d *DocumentRepositoryImpl) InsertStructureNodeBatch(ctx context.Context, nodes []*entity.DocumentStructureNode) error {
	modelList := convert.ToDocumentStructureNodeModelList(nodes)
	return d.dbWithContext(ctx).Create(&modelList).Error
}

func (d *DocumentRepositoryImpl) DeleteStructureNodeByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentStructureNode{}).Error
}

func (d *DocumentRepositoryImpl) DeleteStructureNodeBatch(ctx context.Context, documentId int64, nodeIds []int64) error {
	// TODO implement me
	panic("implement me")
}

func (d *DocumentRepositoryImpl) SelectStructureNodeListByDocumentId(ctx context.Context, documentId int64) ([]*entity.DocumentStructureNode, error) {
	var nodes []*entity.DocumentStructureNode
	err := d.dbWithContext(ctx).Model(&model.DocumentStructureNode{}).
		Where("document_id = ?", documentId).Order("id ASC").Find(&nodes).Error
	return nodes, err
}

// ========== 属性相关 ==========

// InsertProfile 插入文档属性
func (d *DocumentRepositoryImpl) InsertProfile(ctx context.Context, profile *entity.DocumentProfile) error {
	if profile == nil {
		return nil
	}
	return d.dbWithContext(ctx).Model(&model.DocumentProfile{}).Create(profile).Error
}

// SelectProfileByDocumentId 根据文档ID查询文档属性
func (d *DocumentRepositoryImpl) SelectProfileByDocumentId(ctx context.Context, documentId int64) (*entity.DocumentProfile, error) {
	profile := new(entity.DocumentProfile)
	err := d.dbWithContext(ctx).Model(&model.DocumentProfile{}).
		Where("document_id = ?", documentId).
		Order("id DESC").
		First(profile).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return profile, err
}

// UpsertProfile 创建或更新文档属性
func (d *DocumentRepositoryImpl) UpsertProfile(ctx context.Context, profile *entity.DocumentProfile) error {
	if profile == nil {
		return nil
	}
	existing, err := d.SelectProfileByDocumentId(ctx, profile.DocumentId)
	if err != nil {
		return err
	}
	if existing == nil || existing.ID == 0 {
		if profile.ID == 0 {
			profile.ID = utils.GetSnowflakeNextID()
		}
		return d.dbWithContext(ctx).Model(&model.DocumentProfile{}).Create(profile).Error
	}
	profile.ID = existing.ID
	return d.dbWithContext(ctx).Model(&model.DocumentProfile{}).
		Where("id = ?", existing.ID).
		Updates(profile).Error
}

// DeleteProfileByDocumentId 根据文档ID删除文档属性
func (d *DocumentRepositoryImpl) DeleteProfileByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentProfile{}).Error
}

// ========== 话题关联相关 ==========

// DeleteTopicDocumentRelationByDocumentId 根据文档ID删除话题关联
func (d *DocumentRepositoryImpl) DeleteTopicDocumentRelationByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.TopicDocumentRelation{}).Error
}
