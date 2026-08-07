package model

import "github.com/swiftbit/know-agent/common"

type DocumentBlock struct {
	common.Model
	DocumentId        int64  `gorm:"column:document_id;type:bigint"`             // 文档ID
	TaskId            int64  `gorm:"column:task_id;type:bigint"`                 // 任务ID
	BlockNo           int    `gorm:"column:block_no;type:int"`                   // 块序号
	BlockType         string `gorm:"column:block_type;type:varchar(50)"`         // 块类型
	ParentBlockId     int64  `gorm:"column:parent_block_id;type:bigint"`         // 父块ID
	SectionPath       string `gorm:"column:section_path;type:varchar(500)"`      // 章节路径
	CanonicalPath     string `gorm:"column:canonical_path;type:varchar(500)"`    // 规范路径
	PageNo            int    `gorm:"column:page_no;type:int"`                    // 页码
	PageRange         string `gorm:"column:page_range;type:varchar(50)"`         // 页码范围
	BboxJSON          string `gorm:"column:bbox_json;type:text"`                 // 边界框JSON
	Text              string `gorm:"column:text;type:text"`                      // 文本内容
	ContentWithWeight string `gorm:"column:content_with_weight;type:text"`       // 加权内容
	TableHTML         string `gorm:"column:table_html;type:text"`                // 表格HTML
	ImageObjectName   string `gorm:"column:image_object_name;type:varchar(255)"` // 图片对象名
	ImageCaption      string `gorm:"column:image_caption;type:text"`             // 图片标题
	MetadataJSON      string `gorm:"column:metadata_json;type:text"`             // 元数据JSON
}
