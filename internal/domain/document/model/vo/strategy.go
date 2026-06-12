package vo

// StrategyRole 策略角色
type StrategyRole = int

const (
	StrategyRoleSplitter StrategyRole = iota + 1
	StrategyRoleParser
	StrategyRoleIndexer
)

func StrategyRoleName(role StrategyRole) string {
	switch role {
	case StrategyRoleSplitter:
		return "切块器"
	case StrategyRoleParser:
		return "解析器"
	case StrategyRoleIndexer:
		return "索引器"
	default:
		return ""
	}
}

// StrategySourceType 策略来源类型
type StrategySourceType = int

const (
	StrategySourceTypeOriginal StrategySourceType = iota + 1
	StrategySourceTypeParsed
)

func StrategySourceTypeName(sourceType StrategySourceType) string {
	switch sourceType {
	case StrategySourceTypeOriginal:
		return "原始"
	case StrategySourceTypeParsed:
		return "已解析"
	default:
		return ""
	}
}

// StrategyStatus 策略状态
type StrategyStatus = int

const (
	StrategyStatusWaitRecommend StrategyStatus = iota + 1 // 待推荐
	StrategyStatusRecommended                             // 已推荐
	StrategyStatusConfirmed                               // 已确认
	StrategyStatusExpired                                 // 已失效
)

func StrategyStatusName(status StrategyStatus) string {
	switch status {
	case StrategyStatusWaitRecommend:
		return "待推荐"
	case StrategyStatusRecommended:
		return "已推荐"
	case StrategyStatusConfirmed:
		return "已确认"
	case StrategyStatusExpired:
		return "已失效"
	default:
		return ""
	}
}

// StrategyType 策略类型
type StrategyType = int

const (
	StrategyTypeSemanticChunk  StrategyType = iota + 1 // 语义切块
	StrategyTypeMarkdownChunk                          // Markdown切块
	StrategyTypeRecursiveChunk                         // 递归切块
)

func StrategyTypeName(st StrategyType) string {
	switch st {
	case StrategyTypeSemanticChunk:
		return "语义切块"
	case StrategyTypeMarkdownChunk:
		return "Markdown切块"
	case StrategyTypeRecursiveChunk:
		return "递归切块"
	default:
		return ""
	}
}
