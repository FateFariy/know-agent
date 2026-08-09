package entity

// TableCell 文档表格单元格实体
type TableCell struct {
	ID             int64   `gorm:"column:id"`               // 主键ID
	DocumentId     int64   `gorm:"column:document_id"`      // 文档ID
	TaskId         int64   `gorm:"column:task_id"`          // 任务ID
	TableId        int64   `gorm:"column:table_id"`         // 表格ID
	RowId          int64   `gorm:"column:row_id"`           // 行ID
	ColumnId       int64   `gorm:"column:column_id"`        // 列ID
	RowNo          int     `gorm:"column:row_no"`           // 行序号
	ColumnNo       int     `gorm:"column:column_no"`        // 列序号
	CellText       string  `gorm:"column:cell_text"`        // 单元格文本
	NumericValue   float64 `gorm:"column:numeric_value"`    // 数值
	SourceRowNo    int     `gorm:"column:source_row_no"`    // 源行号
	SourceColumnNo int     `gorm:"column:source_column_no"` // 源列号
	SourceCellRef  string  `gorm:"column:source_cell_ref"`  // 源单元格引用
	BboxJSON       string  `gorm:"column:bbox_json"`        // 边界框 JSON
	MetadataJSON   string  `gorm:"column:metadata_json"`    // 元数据 JSON
}
