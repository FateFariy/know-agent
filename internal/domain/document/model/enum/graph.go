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
