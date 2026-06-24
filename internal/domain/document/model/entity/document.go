package entity

import (
	"time"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

type Document struct {
	ID                   int64     `gorm:"column:id"`                    // 主键ID
	DocumentName         string    `gorm:"column:document_name"`         // 文档名称
	OriginalFileName     string    `gorm:"column:original_file_name"`    // 原始文件名
	FileType             int       `gorm:"column:file_type"`             // 文件类型
	MimeType             string    `gorm:"column:mime_type"`             // 媒体类型
	FileSize             int64     `gorm:"column:file_size"`             // 文件大小
	StorageType          int       `gorm:"column:storage_type"`          // 存储类型
	BucketName           string    `gorm:"column:bucket_name"`           // 存储桶名称
	ObjectName           string    `gorm:"column:object_name"`           // 对象名称
	ObjectUrl            string    `gorm:"column:object_url"`            // 对象访问地址
	ParseStatus          int       `gorm:"column:parse_status"`          // 解析状态
	StrategyStatus       int       `gorm:"column:strategy_status"`       // 策略状态
	IndexStatus          int       `gorm:"column:index_status"`          // 索引状态
	CharCount            int       `gorm:"column:char_count"`            // 字符数
	TokenCount           int       `gorm:"column:token_count"`           // Token数量
	StructureLevel       int       `gorm:"column:structure_level"`       // 结构化等级
	ContentQualityLevel  int       `gorm:"column:content_quality_level"` // 内容质量等级
	ParseTextPath        string    `gorm:"column:parse_text_path"`       // 解析文本路径
	ParseErrorMsg        string    `gorm:"column:parse_error_msg"`       // 解析错误信息
	KnowledgeScopeCode   string    `gorm:"column:knowledge_scope_code"`  // 知识范围编码
	KnowledgeScopeName   string    `gorm:"column:knowledge_scope_name"`  // 知识范围名称
	BusinessCategory     string    `gorm:"column:business_category"`     // 业务分类
	DocumentTags         string    `gorm:"column:document_tags"`         // 文档标签
	CurrentPlanId        int64     `gorm:"column:current_plan_id"`       // 当前方案ID
	LastParseTaskId      int64     `gorm:"column:last_parse_task_id"`    // 上一次解析任务ID
	StructureNodeCount   int       `gorm:"column:structure_node_count"`  // 结构化节点数
	LastIndexTaskId      int64     `gorm:"column:last_index_task_id"`    // 上一次索引任务ID
	CreateTime           time.Time `gorm:"column:create_time"`           // 创建时间
	UpdateTime           time.Time `gorm:"column:update_time"`           // 更新时间
	OperatorId           int64     `gorm:"-"`                            // 操作人ID
	FileTypeName         string    `gorm:"-"`                            // 文件类型名称
	ParseStatusName      string    `gorm:"-"`                            // 解析状态名称
	StrategyStatusName   string    `gorm:"-"`                            // 策略状态名称
	IndexStatusName      string    `gorm:"-"`                            // 索引状态名称
	LatestTaskId         int64     `gorm:"-"`                            // 最新任务ID
	LatestTaskType       int       `gorm:"-"`                            // 最新任务类型
	LatestTaskTypeName   string    `gorm:"-"`                            // 最新任务类型名称
	LatestTaskStatus     int       `gorm:"-"`                            // 最新任务状态
	LatestTaskStatusName string    `gorm:"-"`                            // 最新任务状态名称
	PlanReady            bool      `gorm:"-"`                            // 方案是否就绪
}

func (d *Document) FillEnumNames() {
	d.FileTypeName = vo.FileTypeName(d.FileType)
	d.ParseStatusName = vo.ParseStatusName(d.ParseStatus)
	d.StrategyStatusName = vo.StrategyStatusName(d.StrategyStatus)
	d.IndexStatusName = vo.IndexStatusName(d.IndexStatus)
}

func (d *Document) FillLatestTaskInfo(task *DocumentTask) {
	d.LatestTaskId = task.ID
	d.LatestTaskType = task.TaskType
	d.LatestTaskTypeName = vo.TaskTypeName(task.TaskType)
	d.LatestTaskStatus = task.TaskStatus
	d.LatestTaskStatusName = vo.TaskStatusName(task.TaskStatus)
}
