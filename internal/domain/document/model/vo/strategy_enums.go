package vo

// ============================================================
// StrategyStatus 策略状态
// ============================================================

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

// ============================================================
// StrategyType 策略类型
// ============================================================

type StrategyType = int

const (
	StrategyTypeStructure StrategyType = iota + 1 // 结构切块
	StrategyTypeRecursive                         // 递归切块
	StrategyTypeSemantic                          // 语义切块
	StrategyTypeLLM                               // 大模型智能切块
)

func StrategyTypeName(st StrategyType) string {
	switch st {
	case StrategyTypeStructure:
		return "结构切块"
	case StrategyTypeRecursive:
		return "递归切块"
	case StrategyTypeSemantic:
		return "语义切块"
	case StrategyTypeLLM:
		return "大模型智能切块"
	default:
		return ""
	}
}

// ============================================================
// StrategyExecuteStatus 策略执行状态
// ============================================================

type StrategyExecuteStatus = int

const (
	StrategyExecuteStatusWaitExecute StrategyExecuteStatus = iota + 1
	StrategyExecuteStatusExecuting
	StrategyExecuteStatusExecuteSuccess
	StrategyExecuteStatusExecuteFailed
	StrategyExecuteStatusSkipped
)

func StrategyExecuteStatusName(status StrategyExecuteStatus) string {
	switch status {
	case StrategyExecuteStatusWaitExecute:
		return "待执行"
	case StrategyExecuteStatusExecuting:
		return "执行中"
	case StrategyExecuteStatusExecuteSuccess:
		return "执行成功"
	case StrategyExecuteStatusExecuteFailed:
		return "执行失败"
	case StrategyExecuteStatusSkipped:
		return "已跳过"
	default:
		return ""
	}
}

// ============================================================
// StrategyPipelineType 策略流水线类型
// ============================================================

type StrategyPipelineType = string

const (
	StrategyPipelineTypeParent StrategyPipelineType = "PARENT"
	StrategyPipelineTypeChild  StrategyPipelineType = "CHILD"
)

func StrategyPipelineTypeName(pipelineType StrategyPipelineType) string {
	switch pipelineType {
	case StrategyPipelineTypeParent:
		return "父块流水线"
	case StrategyPipelineTypeChild:
		return "子块流水线"
	default:
		return ""
	}
}

// ============================================================
// StrategyRole 策略角色
// ============================================================

type StrategyRole = int

const (
	StrategyRolePrimary StrategyRole = iota + 1
	StrategyRoleOptimize
	StrategyRoleFallback
	StrategyRoleEnhance
)

func StrategyRoleName(role StrategyRole) string {
	switch role {
	case StrategyRolePrimary:
		return "主策略"
	case StrategyRoleOptimize:
		return "优化策略"
	case StrategyRoleFallback:
		return "回退策略"
	case StrategyRoleEnhance:
		return "增强策略"
	default:
		return ""
	}
}

// ============================================================
// StrategySourceType 策略来源类型
// ============================================================

type StrategySourceType = int

const (
	StrategySourceTypeSystemRecommend StrategySourceType = iota + 1
	StrategySourceTypeUserAdd
	StrategySourceTypeUserKeep
)

func StrategySourceTypeName(sourceType StrategySourceType) string {
	switch sourceType {
	case StrategySourceTypeSystemRecommend:
		return "系统推荐"
	case StrategySourceTypeUserAdd:
		return "用户添加"
	case StrategySourceTypeUserKeep:
		return "用户保留"
	default:
		return ""
	}
}
