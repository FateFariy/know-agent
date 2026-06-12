package vo

// LogLevel 日志级别
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
