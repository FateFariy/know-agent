package model

import (
	"github.com/swiftbit/know-agent/common"
)

type Document struct {
	common.Model
	DocumentName        string `gorm:"column:document_name;type:varchar(255)"`       // 文档名称
	OriginalFileName    string `gorm:"column:original_file_name;type:varchar(255)"`  // 原始文件名
	FileType            int    `gorm:"column:file_type;type:int"`                    // 文件类型
	MimeType            string `gorm:"column:mime_type;type:varchar(100)"`           // MIME类型
	FileSize            int64  `gorm:"column:file_size;type:bigint"`                 // 文件大小
	StorageType         int    `gorm:"column:storage_type;type:int"`                 // 存储类型
	BucketName          string `gorm:"column:bucket_name;type:varchar(100)"`         // 存储桶名称
	ObjectName          string `gorm:"column:object_name;type:varchar(255)"`         // 对象名称
	ObjectUrl           string `gorm:"column:object_url;type:varchar(500)"`          // 对象URL
	ParseStatus         int    `gorm:"column:parse_status;type:int"`                 // 解析状态
	StrategyStatus      int    `gorm:"column:strategy_status;type:int"`              // 策略状态
	IndexStatus         int    `gorm:"column:index_status;type:int"`                 // 索引状态
	CharCount           int    `gorm:"column:char_count;type:int"`                   // 字符数
	TokenCount          int    `gorm:"column:token_count;type:int"`                  // Token数
	StructureLevel      int    `gorm:"column:structure_level;type:int"`              // 结构层级
	ContentQualityLevel int    `gorm:"column:content_quality_level;type:int"`        // 内容质量等级
	ParseTextPath       string `gorm:"column:parse_text_path;type:varchar(500)"`     // 解析文本路径
	ParseErrorMsg       string `gorm:"column:parse_error_msg;type:text"`             // 解析错误信息
	KnowledgeBaseId     int64  `gorm:"column:knowledge_base_id;type:bigint"`         // 知识库ID
	KnowledgeBaseName   string `gorm:"column:knowledge_base_name;type:varchar(255)"` // 知识库名称
	CurrentPlanId       int64  `gorm:"column:current_plan_id;type:bigint"`           // 当前计划ID
	LastParseTaskId     int64  `gorm:"column:last_parse_task_id;type:bigint"`        // 最后解析任务ID
	StructureNodeCount  int    `gorm:"column:structure_node_count;type:int"`         // 结构节点数
	LastIndexTaskId     int64  `gorm:"column:last_index_task_id;type:bigint"`        // 最后索引任务ID
}
