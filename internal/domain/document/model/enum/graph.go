package enum

// GraphPersistenceOutcome 图谱持久化结果枚举
type GraphPersistenceOutcome = string

const (
	GraphPersistenceOutcomeSuccess  GraphPersistenceOutcome = "SUCCESS"
	GraphPersistenceOutcomeFailed   GraphPersistenceOutcome = "FAILED"
	GraphPersistenceOutcomeEmpty    GraphPersistenceOutcome = "EMPTY"
	GraphPersistenceOutcomeDegraded GraphPersistenceOutcome = "DEGRADED"
)

// ComponentOutcome 组件执行结果枚举
type ComponentOutcome = string

const (
	ComponentOutcomeSuccess       ComponentOutcome = "SUCCESS"
	ComponentOutcomeFailed        ComponentOutcome = "FAILED"
	ComponentOutcomeNotApplicable ComponentOutcome = "NOT_APPLICABLE"
)

// DerivedIndexOutcome 衍生索引结果枚举
type DerivedIndexOutcome = string

const (
	DerivedIndexOutcomeSuccess DerivedIndexOutcome = "SUCCESS"
	DerivedIndexOutcomeFailed  DerivedIndexOutcome = "FAILED"
)

// ObservationProjectionOutcome 观察投影结果枚举
type ObservationProjectionOutcome = string

const (
	ObservationProjectionOutcomeSuccess ObservationProjectionOutcome = "SUCCESS"
	ObservationProjectionOutcomeFailed  ObservationProjectionOutcome = "FAILED"
)

// OuterTaskDisposition 外部任务处置枚举
type OuterTaskDisposition = string

const (
	OuterTaskDispositionContinue       OuterTaskDisposition = "CONTINUE"
	OuterTaskDispositionRepairRequired OuterTaskDisposition = "REPAIR_REQUIRED"
	OuterTaskDispositionFailIndexTask  OuterTaskDisposition = "FAIL_INDEX_TASK"
)

// InvocationOutcome 调用结果枚举
type InvocationOutcome = string

const (
	InvocationOutcomeSuccess         InvocationOutcome = "SUCCESS"
	InvocationOutcomeFailed          InvocationOutcome = "FAILED"
	InvocationOutcomeNotCalled       InvocationOutcome = "NOT_CALLED"
	InvocationOutcomeInvalidResponse InvocationOutcome = "INVALID_RESPONSE"
	InvocationOutcomeTransportFailed InvocationOutcome = "TRANSPORT_FAILED"
	InvocationOutcomeNotGraphable    InvocationOutcome = "NOT_GRAPHABLE"
	InvocationOutcomeDisabled        InvocationOutcome = "DISABLED"
	InvocationOutcomeEmpty           InvocationOutcome = "EMPTY"
)

// GraphRagEntityTypes 允许抽取的实体类型（中文法规领域受控列表，模板注入用）
var GraphRagEntityTypes = map[string]struct{}{
	"法律": {}, "条款": {}, "机构": {}, "人员": {}, "行为": {},
	"义务": {}, "处罚": {}, "产品": {}, "类别": {}, "标准": {}, "概念": {},
}

// GraphRagRelationFallback 关系类型兜底值：白名单外一律落该关系类型，保留 predicate_quote 原文谓词
const GraphRagRelationFallback = "RELATED_TO"

// GraphRagRelationWhitelist 受控关系类型白名单（中文法规领域，命中→动态关系类型；未命中→RELATED_TO 兜底）
var GraphRagRelationWhitelist = map[string]struct{}{
	"属于": {}, "包含": {}, "类别编号": {}, "定义": {}, "适用于": {},
	"引用": {}, "应当": {}, "不得": {}, "处以": {}, "关联": {},
}

// NormalizeRelationType 归一化关系类型：白名单内原样返回，否则落 RELATED_TO 兜底
func NormalizeRelationType(rt string) string {
	if rt != "" {
		if _, ok := GraphRagRelationWhitelist[rt]; ok {
			return rt
		}
	}
	return GraphRagRelationFallback
}

// IsValidGraphRelation 校验关系类型是否在受控白名单内
func IsValidGraphRelation(rt string) bool {
	if rt == "" {
		return false
	}
	_, ok := GraphRagRelationWhitelist[rt]
	return ok
}

// IsValidGraphEntityType 校验实体类型是否允许
func IsValidGraphEntityType(t string) bool {
	if t == "" {
		return false
	}
	_, ok := GraphRagEntityTypes[t]
	return ok
}
