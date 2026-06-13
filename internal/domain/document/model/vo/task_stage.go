package vo

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
