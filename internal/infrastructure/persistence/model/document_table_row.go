package model

import "github.com/swiftbit/know-agent/common"

type DocumentTableRow struct {
	common.Model
	DocumentId int64  `gorm:"column:document_id;type:bigint"` // 文档ID
	TaskId     int64  `gorm:"column:task_id;type:bigint"`     // 任务ID
	TableId    int64  `gorm:"column:table_id;type:bigint"`    // 表格ID
	RowNo      int    `gorm:"column:row_no;type:int"`         // 行号
	RowText    string `gorm:"column:row_text;type:text"`      // 行文本
}
