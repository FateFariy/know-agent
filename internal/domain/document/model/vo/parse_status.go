package vo

// ParseStatus 解析状态
type ParseStatus = int

const (
	ParseStatusParsing      ParseStatus = iota + 1 // 解析中
	ParseStatusParseSuccess                        // 解析成功
	ParseStatusParseFailed                         // 解析失败
)

func ParseStatusName(statusName ParseStatus) string {
	switch statusName {
	case ParseStatusParsing:
		return "解析中"
	case ParseStatusParseSuccess:
		return "解析成功"
	case ParseStatusParseFailed:
		return "解析失败"
	default:
		return ""
	}
}
