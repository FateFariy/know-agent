package vo

// OperatorType 操作人类型
type OperatorType = int

const (
	OperatorTypeUnknown OperatorType = iota
	OperatorTypeSystem
	OperatorTypeUser
)
