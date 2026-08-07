package model

import "github.com/swiftbit/know-agent/common"

type DocumentBlock struct {
	common.Model
	DocumentId        int64  `gorm:"column:document_id"`
	TaskId            int64  `gorm:"column:task_id"`
	BlockNo           int    `gorm:"column:block_no"`
	BlockType         string `gorm:"column:block_type"`
	ParentBlockId     int64  `gorm:"column:parent_block_id"`
	SectionPath       string `gorm:"column:section_path"`
	CanonicalPath     string `gorm:"column:canonical_path"`
	PageNo            int    `gorm:"column:page_no"`
	PageRange         string `gorm:"column:page_range"`
	BboxJSON          string `gorm:"column:bbox_json"`
	Text              string `gorm:"column:text"`
	ContentWithWeight string `gorm:"column:content_with_weight"`
	TableHTML         string `gorm:"column:table_html"`
	ImageObjectName   string `gorm:"column:image_object_name"`
	ImageCaption      string `gorm:"column:image_caption"`
	MetadataJSON      string `gorm:"column:metadata_json"`
}
