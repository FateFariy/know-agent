package vo

// TaskStage 任务阶段
type TaskStage = int

const (
	TaskStageFileUpload TaskStage = iota + 1
	TaskStageParse
	TaskStageStrategyRecommend
	TaskStageStrategyConfirm
	TaskStageChunkExecute
	TaskStageVectorBuild
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
	case TaskStageVectorBuild:
		return "向量构建"
	case TaskStageComplete:
		return "完成"
	default:
		return ""
	}
}
