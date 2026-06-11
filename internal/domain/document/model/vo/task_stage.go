package vo

// TaskStage 任务阶段
type TaskStage = int

const (
	TaskStageUnknown TaskStage = iota
	TaskStageFileUpload
	TaskStageParse
	TaskStageStrategyRecommend
	TaskStageStrategyConfirm
	TaskStageChunkExecute
	TaskStageVectorBuild
	TaskStageComplete
)
