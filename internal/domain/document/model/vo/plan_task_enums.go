package vo

// ============================================================
// PlanSource 方案来源
// ============================================================

type PlanSource = int

const (
	PlanSourceSystemRecommend PlanSource = iota + 1
	PlanSourceUserAdjust
)

func PlanSourceName(source PlanSource) string {
	switch source {
	case PlanSourceSystemRecommend:
		return "系统推荐"
	case PlanSourceUserAdjust:
		return "用户调整"
	default:
		return ""
	}
}

// ============================================================
// PlanStatus 方案状态
// ============================================================

type PlanStatus = int

const (
	PlanStatusConfirmed PlanStatus = iota + 1
	PlanStatusDiscarded
	PlanStatusWaitConfirm // 待确认
	PlanStatusExecuted    // 已执行
)

func PlanStatusName(status PlanStatus) string {
	switch status {
	case PlanStatusConfirmed:
		return "已确认"
	case PlanStatusDiscarded:
		return "已废弃"
	case PlanStatusWaitConfirm:
		return "待确认"
	case PlanStatusExecuted:
		return "已执行"
	default:
		return ""
	}
}

// ============================================================
// TaskEventType 任务事件类型
// ============================================================

type TaskEventType = int

const (
	TaskEventStart TaskEventType = iota + 1
	TaskEventComplete
	TaskEventFailed
	TaskEventUserConfirm
	TaskEventUserAdjust
	TaskEventRecommendStrategy
)

func TaskEventTypeName(et TaskEventType) string {
	switch et {
	case TaskEventStart:
		return "开始"
	case TaskEventComplete:
		return "完成"
	case TaskEventFailed:
		return "失败"
	case TaskEventUserConfirm:
		return "用户确认"
	case TaskEventUserAdjust:
		return "用户调整"
	case TaskEventRecommendStrategy:
		return "推荐策略"
	default:
		return ""
	}
}

// ============================================================
// TaskStatus 任务状态
// ============================================================

type TaskStatus = int

const (
	TaskStatusNew       TaskStatus = iota + 1 // 新建
	TaskStatusRunning                         // 运行中
	TaskStatusCompleted                       // 完成
	TaskStatusFailed                          // 失败
	TaskStatusSuccess                         // 成功
)

func TaskStatusName(ts TaskStatus) string {
	switch ts {
	case TaskStatusNew:
		return "新建"
	case TaskStatusRunning:
		return "运行中"
	case TaskStatusCompleted:
		return "已完成"
	case TaskStatusFailed:
		return "失败"
	case TaskStatusSuccess:
		return "成功"
	default:
		return ""
	}
}

// ============================================================
// TaskStage 任务阶段
// ============================================================

type TaskStage = int

const (
	TaskStageFileUpload       TaskStage = iota + 1 // 文件上传
	TaskStageContentParse                          // 内容解析
	TaskStageStrategyRoute                         // 策略路由
	TaskStageStrategyConfirm                       // 策略确认
	TaskStageChunkExecute                          // 切块执行
	TaskStageChunkPostProcess                      // 切块后处理
	TaskStageVectorize                             // 向量化
	TaskStageStoreComplete                         // 存储完成
)

func TaskStageName(ts TaskStage) string {
	switch ts {
	case TaskStageFileUpload:
		return "文件上传"
	case TaskStageContentParse:
		return "内容解析"
	case TaskStageStrategyConfirm:
		return "策略确认"
	case TaskStageChunkExecute:
		return "切块执行"
	case TaskStageChunkPostProcess:
		return "切块后处理"
	case TaskStageVectorize:
		return "向量化"
	case TaskStageStoreComplete:
		return "存储完成"
	case TaskStageStrategyRoute:
		return "策略路由"
	default:
		return ""
	}
}

// ============================================================
// TaskType 任务类型
// ============================================================

type TaskType = int

const (
	TaskTypeParseRoute TaskType = iota + 1
	TaskTypeBuildIndex
)

func TaskTypeName(tt TaskType) string {
	switch tt {
	case TaskTypeParseRoute:
		return "解析路由"
	case TaskTypeBuildIndex:
		return "构建索引"
	default:
		return ""
	}
}
