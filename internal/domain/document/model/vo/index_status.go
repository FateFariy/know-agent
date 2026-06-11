package vo

// IndexStatus 索引状态
type IndexStatus = int

const (
	IndexStatusUnknown IndexStatus = iota
	IndexStatusWaitBuild
	IndexStatusBuilding
	IndexStatusBuildSuccess
	IndexStatusBuildFailed
)
