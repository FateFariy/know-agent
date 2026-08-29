package persistence

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	errorx "github.com/swiftbit/know-agent/internal/error"
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence/model"
	"github.com/swiftbit/know-agent/internal/svc"
)

type DocumentRepositoryImpl struct {
	rdb     *redis.Client
	vdb     adapter.VectorIndexer
	storage adapter.Storage
	*transactionManager
}

var _ adapter.DocumentRepository = (*DocumentRepositoryImpl)(nil)

func NewDocumentRepository(svcCtx *svc.ServiceContext, storage adapter.Storage, vdb adapter.VectorIndexer) *DocumentRepositoryImpl {
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
		if err = d.vdb.DeleteByDocumentId(ctx, documentId); err != nil {
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
		if err = d.DeleteParentChunkByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteChunkByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteStructureNodeByDocumentId(ctx, documentId); err != nil {
			return err
		}
		if err = d.DeleteDocumentBlocksByDocumentId(ctx, documentId); err != nil {
			return err
		}
		//if err = d.DeleteTablesByDocumentId(ctx, documentId); err != nil {
		//	return err
		//}
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
	if err := d.dbWithContext(ctx).Model(&model.Document{}).Where(document).First(document).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrDocumentNotFound.Format(documentId)
		}
		return nil, err
	}
	return document, nil
}

// UpdateDocumentById 根据ID更新文档
func (d *DocumentRepositoryImpl) UpdateDocumentById(ctx context.Context, document *entity.Document) error {
	return d.dbWithContext(ctx).Updates(convert.ToDocumentModel(document)).Error
}

// DeleteDocumentById  根据ID删除文档
func (d *DocumentRepositoryImpl) DeleteDocumentById(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("id = ?", documentId).Delete(&model.Document{}).Error
}

// SelectRetrievableDocumentsByIds 根据ID查询可检索的文档
func (d *DocumentRepositoryImpl) SelectRetrievableDocumentsByIds(ctx context.Context, documentIds ...int64) ([]*vo.DocumentMetadata, error) {
	var documents []*vo.DocumentMetadata
	query := d.dbWithContext(ctx).Model(&model.Document{}).
		Where("index_status = ? AND last_index_task_id IS NOT NULL", enum.IndexStatusBuildSuccess)

	if len(documentIds) > 0 {
		query = query.Where("id IN ?", documentIds)
	}
	if err := query.Order("update_time DESC, id DESC").Find(&documents).Error; err != nil {
		return nil, err
	}
	return documents, nil
}

// SelectRetrievableDocumentsByKbIds 根据知识库ID查询可检索的文档
func (d *DocumentRepositoryImpl) SelectRetrievableDocumentsByKbIds(ctx context.Context, kbIds ...int64) ([]*vo.DocumentMetadata, error) {
	var documents []*vo.DocumentMetadata
	query := d.dbWithContext(ctx).Model(&model.Document{}).
		Where("index_status = ? AND last_index_task_id IS NOT NULL", enum.IndexStatusBuildSuccess)

	if len(kbIds) > 0 {
		query = query.Where("knowledge_base_id IN ?", kbIds)
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
	if err := d.dbWithContext(ctx).Model(&model.DocumentTask{}).Where(task).First(task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrTaskNotFound.Format(taskId)
		}
		return nil, err
	}
	return task, nil
}

// SelectLatestTask 根据文档ID获取最新任务
func (d *DocumentRepositoryImpl) SelectLatestTask(ctx context.Context, documentId int64, taskTypes ...int) (*entity.DocumentTask, error) {
	task := &entity.DocumentTask{DocumentId: documentId}
	query := d.dbWithContext(ctx).Model(&model.DocumentTask{}).Where(task).Order("id DESC")
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

// CountTaskByParams 统计任务数量
func (d *DocumentRepositoryImpl) CountTaskByParams(ctx context.Context, documentId int64, taskType int, taskStatus []int) (int64, error) {
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
	if err := d.dbWithContext(ctx).Model(&model.DocumentStrategyPlan{}).Where(plan).First(plan).Error; err != nil {
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
	if err := d.dbWithContext(ctx).Select("plan_version").Where(plan).Order("plan_version DESC").First(plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return plan.PlanVersion, nil
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
func (d *DocumentRepositoryImpl) SelectStepListByPlanId(ctx context.Context, planId int64) (entity.DocumentStrategySteps, error) {
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
	step := &model.DocumentStrategyStep{PlanId: planId, ExecuteStatus: status}
	return d.dbWithContext(ctx).Where("plan_id = ?", planId).Updates(step).Error
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

// UpdateBatchChunkById 根据ID批量更新块
func (d *DocumentRepositoryImpl) UpdateBatchChunkById(ctx context.Context, chunks []*entity.DocumentChunk, fields ...string) error {
	if len(fields) == 0 {
		return errors.New("fields is empty")
	}
	assignments := clause.AssignmentColumns(fields)
	assignments = append(assignments, clause.Assignment{Column: clause.Column{Name: "update_time"}, Value: gorm.Expr("NOW()")})
	return d.dbWithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: assignments,
	}).Create(convert.ToDocumentChunkModelList(chunks)).Error
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
	chunk := &entity.DocumentChunk{ID: chunkId, DocumentId: documentId, TaskId: taskId}
	if err := d.dbWithContext(ctx).Model(&model.DocumentChunk{}).
		Where(chunk).First(chunk).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NewBizError(errorx.ErrDocumentNotFound.Code, "chunk 详情不存在")
		}
		return nil, err
	}
	return chunk, nil
}

// SelectChunkListByParentChunkId 根据父块ID查询块列表
func (d *DocumentRepositoryImpl) SelectChunkListByParentChunkId(ctx context.Context, documentId, taskId, parentChunkId int64) ([]*entity.DocumentChunk, error) {
	var chunks []*entity.DocumentChunk
	if err := d.dbWithContext(ctx).Model(&model.DocumentChunk{}).
		Where("document_id = ? AND task_id = ? AND parent_chunk_id = ?", documentId, taskId, parentChunkId).
		Order("chunk_no ASC").Find(&chunks).Error; err != nil {
		return nil, err
	}
	return chunks, nil
}

// ========== 父块相关 ==========

func (d *DocumentRepositoryImpl) InsertParentChunk(ctx context.Context, block *entity.DocumentParentChunk) error {
	return d.dbWithContext(ctx).Create(convert.ToDocumentParentBlockModel(block)).Error
}

func (d *DocumentRepositoryImpl) InsertParentChunkBatch(ctx context.Context, blocks []*entity.DocumentParentChunk) error {
	return d.dbWithContext(ctx).CreateInBatches(convert.ToDocumentParentBlockModelList(blocks), 100).Error
}

// DeleteParentChunkByDocumentId 根据文档ID删除父块
func (d *DocumentRepositoryImpl) DeleteParentChunkByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentParentChunk{}).Error
}

// SelectParentChunkListByIds 根据父块ID列表查询父块列表
func (d *DocumentRepositoryImpl) SelectParentChunkListByIds(ctx context.Context, ids []int64) ([]*entity.DocumentParentChunk, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var parentBlocks []*entity.DocumentParentChunk
	if err := d.dbWithContext(ctx).Model(&model.DocumentParentChunk{}).
		Where("id IN ?", ids).
		Order("parent_no ASC").
		Find(&parentBlocks).Error; err != nil {
		return nil, err
	}
	return parentBlocks, nil
}

// SelectParentChunkById 根据父块ID查询父块详情
func (d *DocumentRepositoryImpl) SelectParentChunkById(ctx context.Context, blockId, documentId, taskId int64) (*entity.DocumentParentChunk, error) {
	parentBlock := &entity.DocumentParentChunk{ID: blockId, DocumentId: documentId, TaskId: taskId}
	if err := d.dbWithContext(ctx).Model(&model.DocumentParentChunk{}).Where(parentBlock).First(parentBlock).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NewBizError(errorx.ErrDocumentNotFound.Code, "父块详情不存在")
		}
		return nil, err
	}
	return parentBlock, nil
}

// SelectChunks 根据条件查询块列表
func (d *DocumentRepositoryImpl) SelectChunks(ctx context.Context, where map[string]any) ([]*entity.DocumentChunk, error) {
	var chunks []*entity.DocumentChunk
	if err := d.dbWithContext(ctx).Model(&model.DocumentChunk{}).
		Where(where).
		Order("chunk_no ASC, id ASC").
		Find(&chunks).Error; err != nil {
		return nil, err
	}
	return chunks, nil
}

// ========== 结构节点相关 ==========

// InsertStructureNodeBatch 批量插入结构节点
func (d *DocumentRepositoryImpl) InsertStructureNodeBatch(ctx context.Context, nodes []*entity.StructureNode) error {
	return d.dbWithContext(ctx).Create(convert.ToDocumentStructureNodeModelList(nodes)).Error
}

// DeleteStructureNodeByDocumentId 根据文档ID删除结构节点
func (d *DocumentRepositoryImpl) DeleteStructureNodeByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentStructureNode{}).Error
}

// SelectStructureNodeListByDocumentId 根据文档ID查询结构节点列表
func (d *DocumentRepositoryImpl) SelectStructureNodeListByDocumentId(ctx context.Context, documentId int64) ([]*entity.StructureNode, error) {
	var nodes []*entity.StructureNode
	err := d.dbWithContext(ctx).Model(&model.DocumentStructureNode{}).
		Where("document_id = ?", documentId).Order("node_no ASC, id ASC").Find(&nodes).Error
	return nodes, err
}

// SelectStructureNodeListByTask 根据文档ID和任务ID查询结构节点列表
func (d *DocumentRepositoryImpl) SelectStructureNodeListByTask(ctx context.Context, documentId, taskId int64) ([]*entity.StructureNode, error) {
	var nodes []*entity.StructureNode
	if err := d.dbWithContext(ctx).Model(&model.DocumentStructureNode{}).
		Where("document_id = ? AND parse_task_id = ?", documentId, taskId).
		Order("node_no ASC, id ASC").
		Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// CountStructureNodes 统计结构节点数量
func (d *DocumentRepositoryImpl) CountStructureNodes(ctx context.Context, documentId, taskId int64) (int64, error) {
	var count int64
	if err := d.dbWithContext(ctx).Model(&model.DocumentStructureNode{}).
		Where("document_id = ? AND parse_task_id = ?", documentId, taskId).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ========== 属性相关 ==========

// InsertProfile 插入文档属性
func (d *DocumentRepositoryImpl) InsertProfile(ctx context.Context, profile *entity.DocumentProfile) error {
	return d.dbWithContext(ctx).Model(&model.DocumentProfile{}).Create(convert.ToDocumentProfileModel(profile)).Error
}

// SelectProfileByDocumentId 根据文档ID查询文档属性
func (d *DocumentRepositoryImpl) SelectProfileByDocumentId(ctx context.Context, documentId int64) (*entity.DocumentProfile, error) {
	profile := &entity.DocumentProfile{DocumentId: documentId}
	err := d.dbWithContext(ctx).Model(&model.DocumentProfile{}).
		Where(profile).
		Order("id DESC").
		First(profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrDocumentProfileNotFound
		}
		return nil, err
	}
	return profile, nil
}

// SaveProfile 创建或更新文档属性
func (d *DocumentRepositoryImpl) SaveProfile(ctx context.Context, profile *entity.DocumentProfile) error {
	profileModel := convert.ToDocumentProfileModel(profile)
	if profile.ID != 0 {
		return d.dbWithContext(ctx).Where("id = ?", profile.ID).Updates(profileModel).Error
	}
	return d.dbWithContext(ctx).Create(profileModel).Error
}

// DeleteProfileByDocumentId 根据文档ID删除文档属性
func (d *DocumentRepositoryImpl) DeleteProfileByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentProfile{}).Error
}

// SelectDocumentProfiles 查询所有文档属性
func (d *DocumentRepositoryImpl) SelectDocumentProfiles(ctx context.Context) ([]*entity.DocumentProfile, error) {
	var profiles []*entity.DocumentProfile
	if err := d.dbWithContext(ctx).Model(&model.DocumentProfile{}).
		Where("profile_status = ?", 2).
		Find(&profiles).Error; err != nil {
		return nil, err
	}
	return profiles, nil
}

// SelectDocumentProfilesByDocIds 根据文档ID列表查询文档属性
func (d *DocumentRepositoryImpl) SelectDocumentProfilesByDocIds(ctx context.Context, documentIds []int64) ([]*entity.DocumentProfile, error) {
	var profiles []*entity.DocumentProfile
	if err := d.dbWithContext(ctx).Model(&model.DocumentProfile{}).
		Where("document_id IN ?", documentIds).
		Where("profile_status = ?", 2).
		Find(&profiles).Error; err != nil {
		return nil, err
	}
	return profiles, nil
}

// ========== 话题关联相关 ==========

// DeleteTopicDocumentRelationByDocumentId 根据文档ID删除话题关联
func (d *DocumentRepositoryImpl) DeleteTopicDocumentRelationByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.KnowledgeTopicDocumentRelation{}).Error
}

// InsertDocumentBlockBatch 批量插入文档块
func (d *DocumentRepositoryImpl) InsertDocumentBlockBatch(ctx context.Context, blocks []*entity.DocumentBlock) error {
	if len(blocks) == 0 {
		return nil
	}
	return d.dbWithContext(ctx).CreateInBatches(convert.ToDocumentBlockModelList(blocks), 100).Error
}

// SelectDocumentBlocksByTask 根据文档ID和任务ID查询文档块列表
func (d *DocumentRepositoryImpl) SelectDocumentBlocksByTask(ctx context.Context, documentId, taskId int64) (entity.DocumentBlocks, error) {
	var blocks []*entity.DocumentBlock
	if err := d.dbWithContext(ctx).Model(&model.DocumentBlock{}).
		Where("document_id = ? AND task_id = ?", documentId, taskId).
		Order("block_no ASC, id ASC").
		Find(&blocks).Error; err != nil {
		return nil, err
	}
	return blocks, nil
}

// DeleteDocumentBlocksByTask 根据文档ID和任务ID删除文档块
func (d *DocumentRepositoryImpl) DeleteDocumentBlocksByTask(ctx context.Context, documentId, taskId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ? AND task_id = ?", documentId, taskId).
		Delete(&model.DocumentBlock{}).Error
}

// DeleteDocumentBlocksByDocumentId 根据文档ID删除文档块
func (d *DocumentRepositoryImpl) DeleteDocumentBlocksByDocumentId(ctx context.Context, documentId int64) error {
	return d.dbWithContext(ctx).Where("document_id = ?", documentId).Delete(&model.DocumentBlock{}).Error
}

// CountDocumentBlocks 统计文档块数量
func (d *DocumentRepositoryImpl) CountDocumentBlocks(ctx context.Context, documentId, taskId int64) (int64, error) {
	var count int64
	if err := d.dbWithContext(ctx).Model(&model.DocumentBlock{}).
		Where("document_id = ? AND task_id = ?", documentId, taskId).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SelectDocumentBlocksWithLimit 查询文档块列表（带限制）
func (d *DocumentRepositoryImpl) SelectDocumentBlocksWithLimit(ctx context.Context, documentId, taskId int64, limit int) ([]*entity.DocumentBlock, error) {
	var blocks []*entity.DocumentBlock
	if err := d.dbWithContext(ctx).Model(&model.DocumentBlock{}).
		Where("document_id = ? AND task_id = ?", documentId, taskId).
		Order("block_no ASC, id ASC").
		Limit(limit).
		Find(&blocks).Error; err != nil {
		return nil, err
	}
	return blocks, nil
}

// SelectDocumentBlockPageNumbers 查询文档块中不重复的页码列表
func (d *DocumentRepositoryImpl) SelectDocumentBlockPageNumbers(ctx context.Context, documentId, taskId int64) ([]int, error) {
	var pageNumbers []int
	if err := d.dbWithContext(ctx).Model(&model.DocumentBlock{}).
		Where("document_id = ? AND task_id = ? AND page_no IS NOT NULL AND bbox_json IS NOT NULL AND bbox_json != ''", documentId, taskId).
		Distinct("page_no").
		Order("page_no ASC").
		Pluck("page_no", &pageNumbers).Error; err != nil {
		return nil, err
	}
	return pageNumbers, nil
}

// ========== 知识库统计相关 ==========

// CountDocumentsByKbIds 按知识库ID列表统计文档数量
func (d *DocumentRepositoryImpl) CountDocumentsByKbIds(ctx context.Context, knowledgeBaseIds []int64) (map[int64]int64, error) {
	return d.countByKnowledgeBaseIds(ctx, knowledgeBaseIds, nil)
}

// CountRetrievableDocumentsByKbIds 按知识库ID列表统计可检索文档数量
func (d *DocumentRepositoryImpl) CountRetrievableDocumentsByKbIds(ctx context.Context, knowledgeBaseIds []int64) (map[int64]int64, error) {
	return d.countByKnowledgeBaseIds(ctx, knowledgeBaseIds, func(query *gorm.DB) *gorm.DB {
		return query.Where("index_status = ? AND last_index_task_id IS NOT NULL", enum.IndexStatusBuildSuccess)
	})
}

// countByKnowledgeBaseIds 通用统计方法，支持自定义查询条件
func (d *DocumentRepositoryImpl) countByKnowledgeBaseIds(ctx context.Context, knowledgeBaseIds []int64, applyCondition func(*gorm.DB) *gorm.DB) (map[int64]int64, error) {
	if len(knowledgeBaseIds) == 0 {
		return map[int64]int64{}, nil
	}

	type result struct {
		KnowledgeBaseId int64 `gorm:"column:knowledge_base_id"`
		Count           int64 `gorm:"column:count"`
	}

	var results []result
	query := d.dbWithContext(ctx).Model(&model.Document{}).
		Select("knowledge_base_id, COUNT(*) as count").
		Where("knowledge_base_id IN ?", knowledgeBaseIds)

	if applyCondition != nil {
		query = applyCondition(query)
	}

	if err := query.Group("knowledge_base_id").Find(&results).Error; err != nil {
		return nil, err
	}

	countMap := make(map[int64]int64, len(results))
	for _, r := range results {
		countMap[r.KnowledgeBaseId] = r.Count
	}
	return countMap, nil
}
