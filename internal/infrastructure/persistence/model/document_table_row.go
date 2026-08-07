package model

import "github.com/swiftbit/know-agent/common"

type DocumentTableRow struct {
	common.Model
	DocumentId int64  `gorm:"column:document_id"`
	TaskId     int64  `gorm:"column:task_id"`
	TableId    int64  `gorm:"column:table_id"`
	RowNo      int    `gorm:"column:row_no"`
	RowText    string `gorm:"column:row_text"`
}
