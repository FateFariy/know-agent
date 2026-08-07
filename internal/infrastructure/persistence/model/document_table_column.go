package model

import "github.com/swiftbit/know-agent/common"

type DocumentTableColumn struct {
	common.Model
	DocumentId     int64  `gorm:"column:document_id"`
	TaskId         int64  `gorm:"column:task_id"`
	TableId        int64  `gorm:"column:table_id"`
	ColumnNo       int    `gorm:"column:column_no"`
	ColumnName     string `gorm:"column:column_name"`
	NormalizedName string `gorm:"column:normalized_name"`
	ValueType      string `gorm:"column:value_type"`
}
