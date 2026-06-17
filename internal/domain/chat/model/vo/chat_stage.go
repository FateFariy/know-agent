package vo

type ChatStage = string

const (
	ChatStageRewrite   ChatStage = "rewrite"
	ChatStageRetrieval ChatStage = "retrieval"
	ChatStageSummary   ChatStage = "summary"
)
