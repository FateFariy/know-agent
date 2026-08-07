package model

import "github.com/swiftbit/know-agent/common"

type DocumentParseArtifact struct {
	common.Model
	DocumentId    int64  `gorm:"column:document_id;type:bigint"`         // 文档ID
	TaskId        int64  `gorm:"column:task_id;type:bigint"`             // 任务ID
	ArtifactType  string `gorm:"column:artifact_type;type:varchar(50)"`  // 制品类型
	ObjectName    string `gorm:"column:object_name;type:varchar(512)"`   // 对象存储键
	ContentHash   string `gorm:"column:content_hash;type:varchar(128)"`  // 内容哈希
	ParserName    string `gorm:"column:parser_name;type:varchar(100)"`   // 解析器名称
	ParserVersion string `gorm:"column:parser_version;type:varchar(50)"` // 解析器版本
}
