package vo

// VectorStatus 向量状态
type VectorStatus = int

const (
	VectorStatusUnknown VectorStatus = iota
	VectorStatusPending
	VectorStatusBuilding
	VectorStatusBuilt
	VectorStatusFailed
)
