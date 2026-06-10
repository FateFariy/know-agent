package vo

import "time"

// IngestionPipelineVO 数据摄取管道视图对象
type IngestionPipelineVO struct {
	ID          string                    // 管道ID
	Name        string                    // 管道名称
	Description string                    // 管道描述
	CreatedBy   string                    // 创建人
	Nodes       []IngestionPipelineNodeVO // 管道节点列表
	CreateTime  time.Time                 // 创建时间
	UpdateTime  time.Time                 // 更新时间
}

// IngestionPipelineNodeVO 管道节点视图对象（关联子VO，与原Java依赖保持一致）
type IngestionPipelineNodeVO struct{}
