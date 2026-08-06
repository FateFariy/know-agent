package persistence

import (
	"context"

	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/internal/convert"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/infrastructure/persistence/model"
	"github.com/swiftbit/know-agent/internal/svc"
)

type TableRepositoryImpl struct {
	*transactionManager
}

var _ adapter.TableRepository = (*TableRepositoryImpl)(nil)

func NewTableRepository(svcCtx *svc.ServiceContext) *TableRepositoryImpl {
	return &TableRepositoryImpl{
		transactionManager: &transactionManager{db: svcCtx.Db},
	}
}

// InsertTable 插入表格
func (t *TableRepositoryImpl) InsertTable(ctx context.Context, table *entity.DocumentTable) error {
	return t.dbWithContext(ctx).Create(convert.ToDocumentTableModel(table)).Error
}

// InsertTableColumnBatch 批量插入表格列
func (t *TableRepositoryImpl) InsertTableColumnBatch(ctx context.Context, columns []*entity.DocumentTableColumn) error {
	if len(columns) == 0 {
		return nil
	}
	return t.dbWithContext(ctx).CreateInBatches(convert.ToDocumentTableColumnModelList(columns), 100).Error
}

// InsertTableRowBatch 批量插入表格行
func (t *TableRepositoryImpl) InsertTableRowBatch(ctx context.Context, rows []*entity.DocumentTableRow) error {
	if len(rows) == 0 {
		return nil
	}
	return t.dbWithContext(ctx).CreateInBatches(convert.ToDocumentTableRowModelList(rows), 100).Error
}

// InsertTableCellBatch 批量插入表格单元格
func (t *TableRepositoryImpl) InsertTableCellBatch(ctx context.Context, cells []*entity.DocumentTableCell) error {
	if len(cells) == 0 {
		return nil
	}
	return t.dbWithContext(ctx).CreateInBatches(convert.ToDocumentTableCellModelList(cells), 100).Error
}

// SelectTableById 根据ID查询表格
func (t *TableRepositoryImpl) SelectTableById(ctx context.Context, tableId int64) (*entity.DocumentTable, error) {
	table := &entity.DocumentTable{ID: tableId}
	if err := t.dbWithContext(ctx).Model(&model.DocumentTable{}).Where(table).First(table).Error; err != nil {
		return nil, err
	}
	return table, nil
}

// SelectTablesByTask 根据文档ID和任务ID查询表格列表
func (t *TableRepositoryImpl) SelectTablesByTask(ctx context.Context, documentId, taskId int64) ([]*entity.DocumentTable, error) {
	var tables []*entity.DocumentTable
	if err := t.dbWithContext(ctx).Model(&model.DocumentTable{}).
		Where("document_id = ? AND task_id = ?", documentId, taskId).
		Order("table_no ASC").
		Find(&tables).Error; err != nil {
		return nil, err
	}
	return tables, nil
}

// SelectTableColumnsByTableId 根据表格ID查询列列表
func (t *TableRepositoryImpl) SelectTableColumnsByTableId(ctx context.Context, tableId int64) ([]*entity.DocumentTableColumn, error) {
	var columns []*entity.DocumentTableColumn
	if err := t.dbWithContext(ctx).Model(&model.DocumentTableColumn{}).
		Where("table_id = ?", tableId).
		Order("column_no ASC").
		Find(&columns).Error; err != nil {
		return nil, err
	}
	return columns, nil
}

// SelectTableRowsByTableId 根据表格ID查询行列表
func (t *TableRepositoryImpl) SelectTableRowsByTableId(ctx context.Context, tableId int64) ([]*entity.DocumentTableRow, error) {
	var rows []*entity.DocumentTableRow
	if err := t.dbWithContext(ctx).Model(&model.DocumentTableRow{}).
		Where("table_id = ?", tableId).
		Order("row_no ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SelectTableCellsByTableId 根据表格ID查询单元格列表
func (t *TableRepositoryImpl) SelectTableCellsByTableId(ctx context.Context, tableId int64) ([]*entity.DocumentTableCell, error) {
	var cells []*entity.DocumentTableCell
	if err := t.dbWithContext(ctx).Model(&model.DocumentTableCell{}).
		Where("table_id = ?", tableId).
		Order("row_no ASC, column_no ASC").
		Find(&cells).Error; err != nil {
		return nil, err
	}
	return cells, nil
}

// DeleteTableDetailByTableIds 根据表格ID列表删除表格列、行、单元格
func (t *TableRepositoryImpl) DeleteTableDetailByTableIds(ctx context.Context, tableIds []int64) error {
	if len(tableIds) == 0 {
		return nil
	}
	return t.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("table_id IN ?", tableIds).
			Delete(&model.DocumentTableCell{}).Error; err != nil {
			return err
		}
		if err := tx.Where("table_id IN ?", tableIds).
			Delete(&model.DocumentTableRow{}).Error; err != nil {
			return err
		}
		return tx.Where("table_id IN ?", tableIds).
			Delete(&model.DocumentTableColumn{}).Error
	})
}

// DeleteTablesByTask 根据文档ID和任务ID删除表格
func (t *TableRepositoryImpl) DeleteTablesByTask(ctx context.Context, documentId, taskId int64) error {
	var tableIds []int64
	return t.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := t.dbWithContext(ctx).Model(&model.DocumentTable{}).
			Where("document_id = ? AND task_id = ?", documentId, taskId).
			Pluck("id", &tableIds).Error; err != nil {
			return err
		}
		if len(tableIds) > 0 {
			if err := t.DeleteTableDetailByTableIds(ctx, tableIds); err != nil {
				return err
			}
		}
		return tx.Where("document_id = ? AND task_id = ?", documentId, taskId).
			Delete(&model.DocumentTable{}).Error
	})
}

// DeleteTablesByDocumentId 根据文档ID删除表格
func (t *TableRepositoryImpl) DeleteTablesByDocumentId(ctx context.Context, documentId int64) error {
	var tableIds []int64
	return t.dbWithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := t.dbWithContext(ctx).Model(&model.DocumentTable{}).
			Where("document_id = ?", documentId).
			Pluck("id", &tableIds).Error; err != nil {
			return err
		}
		if len(tableIds) > 0 {
			if err := t.DeleteTableDetailByTableIds(ctx, tableIds); err != nil {
				return err
			}
		}
		return tx.Where("document_id = ?", documentId).
			Delete(&model.DocumentTable{}).Error
	})
}

// CountTables 统计表格数量
func (t *TableRepositoryImpl) CountTables(ctx context.Context, documentId, taskId int64) (int64, error) {
	var count int64
	if err := t.dbWithContext(ctx).Model(&model.DocumentTable{}).
		Where("document_id = ? AND task_id = ?", documentId, taskId).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// SelectTablePageNumbers 查询表格中不重复的页码列表
func (t *TableRepositoryImpl) SelectTablePageNumbers(ctx context.Context, documentId, taskId int64) ([]int, error) {
	var pageNumbers []int
	if err := t.dbWithContext(ctx).Model(&model.DocumentTable{}).
		Where("document_id = ? AND task_id = ? AND page_no IS NOT NULL AND bbox_json IS NOT NULL AND bbox_json != ''", documentId, taskId).
		Distinct("page_no").
		Order("page_no ASC").
		Pluck("page_no", &pageNumbers).Error; err != nil {
		return nil, err
	}
	return pageNumbers, nil
}
