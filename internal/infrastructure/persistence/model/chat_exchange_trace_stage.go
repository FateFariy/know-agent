package model

import (
	"time"

	"github.com/swiftbit/know-agent/common"
)

type ChatExchangeTraceStage struct {
	common.Model
	ConversationId string     `gorm:"column:conversation_id"`
	ExchangeId     int64      `gorm:"column:exchange_id"`
	TraceId        string     `gorm:"column:trace_id"`
	StageCode      string     `gorm:"column:stage_code"`
	StageName      string     `gorm:"column:stage_name"`
	StageOrder     int        `gorm:"column:stage_order"`
	StageLevel     int        `gorm:"column:stage_level"`
	ParentStageId  int64      `gorm:"column:parent_stage_id"`
	ExecutionMode  string     `gorm:"column:execution_mode"`
	StageState     int        `gorm:"column:stage_state"`
	StartTime      *time.Time `gorm:"column:start_time"`
	EndTime        *time.Time `gorm:"column:end_time"`
	DurationMs     int64      `gorm:"column:duration_ms"`
	SummaryText    *string    `gorm:"column:summary_text"`
	ErrorMessage   *string    `gorm:"column:error_message"`
	SnapshotJson   *string    `gorm:"column:snapshot_json"`
}
