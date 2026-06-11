package vo

// StrategyRole 策略角色
type StrategyRole = int

const (
	StrategyRoleUnknown StrategyRole = iota
	StrategyRoleSplitter
	StrategyRoleParser
	StrategyRoleIndexer
)

// StrategySourceType 策略来源类型
type StrategySourceType = int

const (
	StrategySourceTypeUnknown StrategySourceType = iota
	StrategySourceTypeOriginal
	StrategySourceTypeParsed
)

// StrategyStatus 策略状态
type StrategyStatus = int

const (
	StrategyStatusUnknown       StrategyStatus = iota // 未知状态
	StrategyStatusWaitRecommend                       // 待推荐
	StrategyStatusRecommended                         // 已推荐
	StrategyStatusConfirmed                           // 已确认
	StrategyStatusExpired                             // 已失效
)

// StrategyType 策略类型
type StrategyType = int

const (
	StrategyTypeUnknown StrategyType = iota
	StrategyTypeSemanticChunk
	StrategyTypeMarkdownChunk
	StrategyTypeRecursiveChunk
)
