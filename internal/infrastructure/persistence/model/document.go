package model

import (
	"github.com/swiftbit/know-agent/common"
)

type Document struct {
	common.Model
	DocumentName        string  `gorm:"column:document_name"`         // 文档名称
	OriginalFileName    string  `gorm:"column:original_file_name"`    // 原始文件名
	FileType            int     `gorm:"column:file_type"`             // 文件类型
	MimeType            string  `gorm:"column:mime_type"`             // 媒体类型
	FileSize            int64   `gorm:"column:file_size"`             // 文件大小
	StorageType         int     `gorm:"column:storage_type"`          // 存储类型
	BucketName          string  `gorm:"column:bucket_name"`           // 存储桶名称
	ObjectName          string  `gorm:"column:object_name"`           // 对象名称
	ObjectUrl           string  `gorm:"column:object_url"`            // 对象访问地址
	ParseStatus         int     `gorm:"column:parse_status"`          // 解析状态
	StrategyStatus      int     `gorm:"column:strategy_status"`       // 策略状态
	IndexStatus         int     `gorm:"column:index_status"`          // 索引状态
	CharCount           int     `gorm:"column:char_count"`            // 字符数
	TokenCount          int     `gorm:"column:token_count"`           // Token数量
	StructureLevel      int     `gorm:"column:structure_level"`       // 结构化等级
	ContentQualityLevel int     `gorm:"column:content_quality_level"` // 内容质量等级
	ParseTextPath       string  `gorm:"column:parse_text_path"`       // 解析文本路径
	ParseErrorMsg       *string `gorm:"column:parse_error_msg"`       // 解析错误信息
	KnowledgeScopeCode  string  `gorm:"column:knowledge_scope_code"`  // 知识范围编码
	KnowledgeScopeName  string  `gorm:"column:knowledge_scope_name"`  // 知识范围名称
	BusinessCategory    string  `gorm:"column:business_category"`     // 业务分类
	DocumentTags        string  `gorm:"column:document_tags"`         // 文档标签
	CurrentPlanId       int64   `gorm:"column:current_plan_id"`       // 当前方案ID
	LastParseTaskId     int64   `gorm:"column:last_parse_task_id"`    // 上一次解析任务ID
	StructureNodeCount  int     `gorm:"column:structure_node_count"`  // 结构化节点数
	LastIndexTaskId     int64   `gorm:"column:last_index_task_id"`    // 上一次索引任务ID
}
