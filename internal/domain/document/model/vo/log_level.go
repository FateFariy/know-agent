package vo

// LogLevel 日志级别
type LogLevel = int

const (
	LogLevelInfo LogLevel = iota
	LogLevelWarn
	LogLevelError
)
