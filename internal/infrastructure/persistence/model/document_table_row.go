package model

import "github.com/swiftbit/know-agent/common"

type DocumentTableRow struct {
	common.Model
	DocumentID int64  `gorm:"column:document_id"`
	TaskID     int64  `gorm:"column:task_id"`
	TableID    int64  `gorm:"column:table_id"`
	RowNo      int    `gorm:"column:row_no"`
	RowText    string `gorm:"column:row_text"`
}
