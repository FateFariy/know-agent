package vo

// ============================================================
// LogLevel 日志级别
// ============================================================

type LogLevel = int

const (
	LogLevelInfo  LogLevel = iota + 1 // INFO
	LogLevelWarn                      // WARN
	LogLevelError                     // ERROR
)

func LogLevelName(ll LogLevel) string {
	switch ll {
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return ""
	}
}

// ============================================================
// OperatorType 操作人类型
// ============================================================

type OperatorType = int

const (
	OperatorTypeUnknown OperatorType = iota
	OperatorTypeSystem
	OperatorTypeUser
)

// ============================================================
// StorageType 存储类型
// ============================================================

type StorageType = int

const (
	StorageTypeUnknown StorageType = iota
	StorageTypeMINIO
)

// ============================================================
// TriggerSource 触发来源
// ============================================================

type TriggerSource = int

const (
	TriggerSourceUnknown TriggerSource = iota
	TriggerSourceSystem
	TriggerSourceUser
)
