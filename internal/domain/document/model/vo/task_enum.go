package vo

// TaskEventType 任务事件类型
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

// TaskStatus 任务状态
type TaskStatus = int

const (
	TaskStatusNew TaskStatus = iota + 1
	TaskStatusRunning
	TaskStatusCompleted
	TaskStatusFailed
	TaskStatusSuccess // 成功
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

// TaskStage 任务阶段
type TaskStage = int

const (
	TaskStageFileUpload TaskStage = iota + 1
	TaskStageParse
	TaskStageStrategyRecommend
	TaskStageStrategyConfirm
	TaskStageChunkExecute
	TaskStageChunkPostProcess
	TaskStageVectorBuild
	TaskStageVectorize
	TaskStageStoreComplete
	TaskStageStrategyRoute
	TaskStageContentParse
	TaskStageComplete
)

func TaskStageName(ts TaskStage) string {
	switch ts {
	case TaskStageFileUpload:
		return "文件上传"
	case TaskStageParse:
		return "解析"
	case TaskStageStrategyRecommend:
		return "策略推荐"
	case TaskStageStrategyConfirm:
		return "策略确认"
	case TaskStageChunkExecute:
		return "切块执行"
	case TaskStageChunkPostProcess:
		return "切块后处理"
	case TaskStageVectorBuild:
		return "向量构建"
	case TaskStageVectorize:
		return "向量化"
	case TaskStageStoreComplete:
		return "存储完成"
	case TaskStageStrategyRoute:
		return "策略路由"
	case TaskStageContentParse:
		return "内容解析"
	case TaskStageComplete:
		return "完成"
	default:
		return ""
	}
}

// TaskType 任务类型
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
