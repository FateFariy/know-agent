package vo

// StructureLevel 文档结构等级
type StructureLevel = int

const (
	StructureLevelLow    StructureLevel = iota + 1 // 低结构化
	StructureLevelMedium                           // 中结构化
	StructureLevelHigh                             // 高结构化
)

func StructureLevelName(level StructureLevel) string {
	switch level {
	case StructureLevelLow:
		return "低结构化"
	case StructureLevelMedium:
		return "中结构化"
	case StructureLevelHigh:
		return "高结构化"
	default:
		return ""
	}
}
