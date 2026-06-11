package vo

// ParseStatus 解析状态
type ParseStatus = int

const (
	ParseStatusUnknown ParseStatus = iota
	ParseStatusParsing
	ParseStatusParseSuccess
	ParseStatusParseFailed
)
