package vo

// GraphRagBuildResult GraphRAG 构建结果
type GraphRagBuildResult struct {
	EntityCount                  int                          `json:"entityCount"`
	RelationCount                int                          `json:"relationCount"`
	EvidenceCount                int                          `json:"evidenceCount"`
	CommunityCount               int                          `json:"communityCount"`
	GraphPersistenceOutcome      GraphPersistenceOutcome      `json:"graphPersistenceOutcome"`
	GraphPersistenceReason       string                       `json:"graphPersistenceReason"`
	KgCommitted                  bool                         `json:"kgCommitted"`
	TypedIndexOutcome            ComponentOutcome             `json:"typedIndexOutcome"`
	CrossDocumentIndexOutcome    ComponentOutcome             `json:"crossDocumentIndexOutcome"`
	DerivedIndexOutcome          DerivedIndexOutcome          `json:"derivedIndexOutcome"`
	ObservationProjectionOutcome ObservationProjectionOutcome `json:"observationProjectionOutcome"`
	OuterTaskDisposition         OuterTaskDisposition         `json:"outerTaskDisposition"`
	PythonInvocationOutcome      InvocationOutcome            `json:"pythonInvocationOutcome"`
	AdvisorInvocationOutcome     InvocationOutcome            `json:"advisorInvocationOutcome"`
	PythonExtractionStatus       string                       `json:"pythonExtractionStatus"`
	AdvisorReason                string                       `json:"advisorReason"`
	DegradationReasons           []string                     `json:"degradationReasons"`
	ExtractionMetadata           map[string]any               `json:"extractionMetadata"`
	Attempt                      int                          `json:"attempt"`
	MaxAttempts                  int                          `json:"maxAttempts"`
}

// GraphPersistenceOutcome 图谱持久化结果枚举
type GraphPersistenceOutcome = string

const (
	GraphPersistenceOutcomeSuccess GraphPersistenceOutcome = "SUCCESS"
	GraphPersistenceOutcomeFailed  GraphPersistenceOutcome = "FAILED"
	GraphPersistenceOutcomeEmpty   GraphPersistenceOutcome = "EMPTY"
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
	InvocationOutcomeSuccess InvocationOutcome = "SUCCESS"
	InvocationOutcomeFailed  InvocationOutcome = "FAILED"
)

// TypedChunk 类型化块接口
type TypedChunk interface {
	GetID() int64
	GetChunkNo() int
	GetChunkText() string
}

// GraphRagFinalization GraphRAG 最终化结果
type GraphRagFinalization struct {
	Result      *GraphRagBuildResult
	TypedChunks []TypedChunk
}

// RaptorBuildResult RAPTOR 构建结果
type RaptorBuildResult struct {
	NodeCount           int    `json:"nodeCount"`
	LevelCount          int    `json:"levelCount"`
	SourceChunkCount    int    `json:"sourceChunkCount"`
	SourceQualityReport string `json:"sourceQualityReport"`
	SavedQualityReport  string `json:"savedQualityReport"`
}

// GraphRagBuildFailureException GraphRAG 构建失败异常
type GraphRagBuildFailureException struct {
	Result *GraphRagBuildResult
	Err    error
}

func (e *GraphRagBuildFailureException) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Result != nil {
		return e.Result.GraphPersistenceReason
	}
	return "GraphRAG build failed"
}

func (e *GraphRagBuildFailureException) Unwrap() error {
	return e.Err
}
