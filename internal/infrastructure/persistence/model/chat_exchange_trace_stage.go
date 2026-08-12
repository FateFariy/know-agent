package model

import (
	"time"

	"github.com/swiftbit/know-agent/common"
)

type ChatExchangeTraceStage struct {
	common.Model
	ConversationId string     `gorm:"column:conversation_id;type:varchar(64)"` // 对话ID
	ExchangeId     int64      `gorm:"column:exchange_id;type:bigint"`          // 交互ID
	TraceId        string     `gorm:"column:trace_id;type:varchar(64)"`        // 追踪ID
	StageCode      string     `gorm:"column:stage_code;type:varchar(50)"`      // 阶段编码
	StageName      string     `gorm:"column:stage_name;type:varchar(100)"`     // 阶段名称
	StageOrder     int        `gorm:"column:stage_order;type:int"`             // 阶段顺序
	StageLevel     int        `gorm:"column:stage_level;type:int"`             // 阶段层级
	ParentStageId  int64      `gorm:"column:parent_stage_id;type:bigint"`      // 父阶段ID
	ExecutionMode  string     `gorm:"column:execution_mode;type:varchar(50)"`  // 执行模式
	StageState     int        `gorm:"column:stage_state;type:tinyint"`         // 阶段状态
	StartTime      *time.Time `gorm:"column:start_time;type:datetime"`         // 开始时间
	EndTime        *time.Time `gorm:"column:end_time;type:datetime"`           // 结束时间
	DurationMs     int64      `gorm:"column:duration_ms;type:bigint"`          // 耗时(ms)
	SummaryText    *string    `gorm:"column:summary_text;type:text"`           // 阶段摘要
	ErrorMessage   *string    `gorm:"column:error_message;type:varchar(500)"`  // 错误信息
	SnapshotJson   *string    `gorm:"column:snapshot_json;type:text"`          // 快照JSON
}
