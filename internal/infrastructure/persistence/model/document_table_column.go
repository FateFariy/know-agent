package model

import "github.com/swiftbit/know-agent/common"

type DocumentTableColumn struct {
	common.Model
	DocumentId     int64  `gorm:"column:document_id;type:bigint"`           // 文档ID
	TaskId         int64  `gorm:"column:task_id;type:bigint"`               // 任务ID
	TableId        int64  `gorm:"column:table_id;type:bigint"`              // 表格ID
	ColumnNo       int    `gorm:"column:column_no;type:int"`                // 列编号
	ColumnName     string `gorm:"column:column_name;type:varchar(255)"`     // 列名称
	NormalizedName string `gorm:"column:normalized_name;type:varchar(255)"` // 标准化列名
	ValueType      string `gorm:"column:value_type;type:varchar(50)"`       // 值类型
}
