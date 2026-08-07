package model

import "github.com/swiftbit/know-agent/common"

type DocumentTable struct {
	common.Model
	DocumentId   int64  `gorm:"column:document_id"`
	TaskId       int64  `gorm:"column:task_id"`
	BlockId      int64  `gorm:"column:block_id"`
	TableNo      int    `gorm:"column:table_no"`
	SectionPath  string `gorm:"column:section_path"`
	PageNo       int    `gorm:"column:page_no"`
	PageRange    string `gorm:"column:page_range"`
	BboxJSON     string `gorm:"column:bbox_json"`
	Title        string `gorm:"column:title"`
	RowCount     int    `gorm:"column:row_count"`
	ColumnCount  int    `gorm:"column:column_count"`
	TableHTML    string `gorm:"column:table_html"`
	MetadataJSON string `gorm:"column:metadata_json"`
}
