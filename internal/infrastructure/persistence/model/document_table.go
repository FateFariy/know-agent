package model

import "github.com/swiftbit/know-agent/common"

type DocumentTable struct {
	common.Model
	DocumentId   int64  `gorm:"column:document_id;type:bigint"`     // 文档ID
	TaskId       int64  `gorm:"column:task_id;type:bigint"`         // 任务ID
	BlockId      int64  `gorm:"column:block_id;type:bigint"`        // 区块ID
	TableNo      int    `gorm:"column:table_no;type:int"`           // 表格编号
	SectionPath  string `gorm:"column:section_path;type:text"`      // 章节路径
	PageNo       int    `gorm:"column:page_no;type:int"`            // 页码
	PageRange    string `gorm:"column:page_range;type:varchar(50)"` // 页面范围
	BboxJson     string `gorm:"column:bbox_json;type:text"`         // 边界框JSON
	Title        string `gorm:"column:title;type:text"`             // 标题
	RowCount     int    `gorm:"column:row_count;type:int"`          // 行数
	ColumnCount  int    `gorm:"column:column_count;type:int"`       // 列数
	TableHtml    string `gorm:"column:table_html;type:text"`        // 表格HTML
	MetadataJson string `gorm:"column:metadata_json;type:text"`     // 元数据JSON
}
