package entity

import (
	"encoding/json"
	"time"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// DocumentTask 文档任务实体
type DocumentTask struct {
	ID                int64              `gorm:"column:id"`                   // 主键ID
	DocumentId        int64              `gorm:"column:document_id"`          // 文档ID
	PlanId            int64              `gorm:"column:plan_id"`              // 方案ID
	SourceParseTaskId int64              `gorm:"column:source_parse_task_id"` // 源解析任务ID
	TaskType          int                `gorm:"column:task_type"`            // 任务类型
	TaskStatus        int                `gorm:"column:task_status"`          // 任务状态
	CurrentStage      int                `gorm:"column:current_stage"`        // 当前阶段
	TriggerSource     int                `gorm:"column:trigger_source"`       // 触发来源
	StrategySnapshot  string             `gorm:"column:strategy_snapshot"`    // 策略快照
	RetryCount        int                `gorm:"column:retry_count"`          // 重试次数
	StartTime         *time.Time         `gorm:"column:start_time"`           // 开始时间
	FinishTime        *time.Time         `gorm:"column:finish_time"`          // 完成时间
	CostMillis        int64              `gorm:"column:cost_millis"`          // 耗时(毫秒)
	ErrorCode         *string            `gorm:"column:error_code"`           // 错误码
	ErrorMsg          *string            `gorm:"column:error_msg"`            // 错误信息
	ExtJson           string             `gorm:"column:ext_json"`             // 扩展JSON
	TaskTypeName      string             `gorm:"-"`                           // 任务类型名称
	TaskStatusName    string             `gorm:"-"`                           // 任务状态名称
	CurrentStageName  string             `gorm:"-"`                           // 当前阶段名称
	Logs              []*DocumentTaskLog `gorm:"-"`                           // 日志
}

func (t *DocumentTask) FillEnumNames() {
	if t == nil {
		return
	}
	t.TaskTypeName = enum.TaskTypeName(t.TaskType)
	t.TaskStatusName = enum.TaskStatusName(t.TaskStatus)
	t.CurrentStageName = enum.TaskStageName(t.CurrentStage)
	slice.ForEach(t.Logs, func(index int, log *DocumentTaskLog) {
		log.FillEnumNames()
	})
}

// ReadGraphRagBuildResult 从任务扩展 JSON 读取 GraphRAG 构建结果
func (t *DocumentTask) ReadGraphRagBuildResult() *vo.GraphRagBuildResult {
	if t == nil || strutil.IsBlank(t.ExtJson) {
		return nil
	}
	var wrapper struct {
		Result *vo.GraphRagBuildResult `json:"graphRagBuild"`
	}
	if err := json.Unmarshal([]byte(t.ExtJson), &wrapper); err != nil {
		logx.Warnf("Ignoring unreadable GraphRAG outcome checkpoint: taskId=%d, message=%v", t.ID, err)
		return nil
	}
	return wrapper.Result
}
