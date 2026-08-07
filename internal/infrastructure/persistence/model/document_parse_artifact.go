package model

import "github.com/swiftbit/know-agent/common"

type DocumentParseArtifact struct {
	common.Model
	DocumentId    int64  `gorm:"column:document_id"`
	TaskId        int64  `gorm:"column:task_id"`
	ArtifactType  string `gorm:"column:artifact_type"`
	ObjectName    string `gorm:"column:object_name"`
	ContentHash   string `gorm:"column:content_hash"`
	ParserName    string `gorm:"column:parser_name"`
	ParserVersion string `gorm:"column:parser_version"`
}
