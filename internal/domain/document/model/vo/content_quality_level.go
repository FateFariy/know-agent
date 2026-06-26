package vo

// ContentQualityLevel 文档内容质量等级
type ContentQualityLevel = int

const (
	ContentQualityLevelLow    ContentQualityLevel = iota + 1 // 低质量
	ContentQualityLevelMedium                                // 中质量
	ContentQualityLevelHigh                                  // 高质量
)

func ContentQualityLevelName(level ContentQualityLevel) string {
	switch level {
	case ContentQualityLevelLow:
		return "低质量"
	case ContentQualityLevelMedium:
		return "中质量"
	case ContentQualityLevelHigh:
		return "高质量"
	default:
		return ""
	}
}
