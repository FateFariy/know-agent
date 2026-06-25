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
	StrategySourceTypeOriginal        StrategySourceType = iota + 1 // 原始
	StrategySourceTypeParsed                                        // 已解析
	StrategySourceTypeSystemRecommend                               // 系统推荐
	StrategySourceTypeUserAdd                                       // 用户添加
)

func StrategySourceTypeName(sourceType StrategySourceType) string {
	switch sourceType {
	case StrategySourceTypeOriginal:
		return "原始"
	case StrategySourceTypeParsed:
		return "已解析"
	case StrategySourceTypeSystemRecommend:
		return "系统推荐"
	case StrategySourceTypeUserAdd:
		return "用户添加"
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
	StrategyTypeStructure StrategyType = iota + 1 // 结构切块
	StrategyTypeRecursive                         // 递归切块
	StrategyTypeSemantic                          // 语义切块
	StrategyTypeLlm                               // 大模型智能切块
	StrategyTypeMarkdown                          // Markdown切块
)

func StrategyTypeName(st StrategyType) string {
	switch st {
	case StrategyTypeStructure:
		return "结构切块"
	case StrategyTypeRecursive:
		return "递归切块"
	case StrategyTypeSemantic:
		return "语义切块"
	case StrategyTypeLlm:
		return "大模型智能切块"
	case StrategyTypeMarkdown:
		return "Markdown切块"
	default:
		return ""
	}
}
