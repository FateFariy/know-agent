package model

import "github.com/swiftbit/know-agent/common"

type DocumentTableCell struct {
	common.Model
	DocumentId     int64   `gorm:"column:document_id;type:bigint"`          // 文档ID
	TaskId         int64   `gorm:"column:task_id;type:bigint"`              // 任务ID
	TableId        int64   `gorm:"column:table_id;type:bigint"`             // 表格ID
	RowId          int64   `gorm:"column:row_id;type:bigint"`               // 行ID
	ColumnId       int64   `gorm:"column:column_id;type:bigint"`            // 列ID
	RowNo          int     `gorm:"column:row_no;type:int"`                  // 行号
	ColumnNo       int     `gorm:"column:column_no;type:int"`               // 列号
	CellText       string  `gorm:"column:cell_text;type:text"`              // 单元格文本
	NumericValue   float64 `gorm:"column:numeric_value;type:decimal(10,2)"` // 数值
	SourceRowNo    int     `gorm:"column:source_row_no;type:int"`           // 源行号
	SourceColumnNo int     `gorm:"column:source_column_no;type:int"`        // 源列号
	SourceCellRef  string  `gorm:"column:source_cell_ref;type:varchar(50)"` // 源单元格引用
	BboxJson       string  `gorm:"column:bbox_json;type:text"`              // 边界框JSON
	MetadataJson   string  `gorm:"column:metadata_json;type:text"`          // 元数据JSON
}
