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

// GraphRagEntityTypes 允许抽取的实体类型（小写归一化后匹配）
var GraphRagEntityTypes = map[string]struct{}{
	"PERSON": {}, "ORG": {}, "ORGANIZATION": {}, "LOC": {}, "LOCATION": {}, "GPE": {},
	"PRODUCT": {}, "EVENT": {}, "WORK_OF_ART": {}, "CONCEPT": {}, "TECH": {}, "PROJECT": {},
}

// GraphRagRelationWhitelist 关系类型白名单（relationType ENUM，落库前严格校验）
var GraphRagRelationWhitelist = map[string]struct{}{
	"RELATED_TO": {}, "BELONGS_TO": {}, "LOCATED_IN": {}, "WORKS_FOR": {}, "FOUNDED": {},
	"PARTICIPATES_IN": {}, "USES": {}, "PRODUCES": {}, "COOPERATES_WITH": {}, "INVESTS_IN": {},
	"ACQUIRED": {}, "COMPETES_WITH": {}, "SERVES": {}, "ALIAS_OF": {}, "BORN_IN": {},
	"GRADUATED_FROM": {}, "REPRESENTATIVE_WORK": {}, "HEADQUARTERED_IN": {},
}

// IsValidGraphRelation 校验关系类型是否在白名单内
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
