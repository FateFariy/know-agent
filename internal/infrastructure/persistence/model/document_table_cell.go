package model

import "github.com/swiftbit/know-agent/common"

type DocumentTableCell struct {
	common.Model
	DocumentID     int64   `gorm:"column:document_id"`
	TaskID         int64   `gorm:"column:task_id"`
	TableID        int64   `gorm:"column:table_id"`
	RowID          int64   `gorm:"column:row_id"`
	ColumnID       int64   `gorm:"column:column_id"`
	RowNo          int     `gorm:"column:row_no"`
	ColumnNo       int     `gorm:"column:column_no"`
	CellText       string  `gorm:"column:cell_text"`
	NumericValue   float64 `gorm:"column:numeric_value"`
	SourceRowNo    int     `gorm:"column:source_row_no"`
	SourceColumnNo int     `gorm:"column:source_column_no"`
	SourceCellRef  string  `gorm:"column:source_cell_ref"`
	BboxJSON       string  `gorm:"column:bbox_json"`
	MetadataJSON   string  `gorm:"column:metadata_json"`
}
