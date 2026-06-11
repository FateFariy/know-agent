package vo

// TriggerSource 触发来源
type TriggerSource = int

const (
	TriggerSourceUnknown TriggerSource = iota
	TriggerSourceSystem
	TriggerSourceUser
)
