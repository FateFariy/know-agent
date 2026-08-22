package vo

import "github.com/swiftbit/know-agent/internal/domain/document/model/enum"

// GraphRagBuildResult GraphRAG 构建结果
type GraphRagBuildResult struct {
	EntityCount                  int                               `json:"entityCount"`
	RelationCount                int                               `json:"relationCount"`
	EvidenceCount                int                               `json:"evidenceCount"`
	CommunityCount               int                               `json:"communityCount"`
	GraphPersistenceOutcome      enum.GraphPersistenceOutcome      `json:"graphPersistenceOutcome"`
	GraphPersistenceReason       string                            `json:"graphPersistenceReason"`
	KgCommitted                  bool                              `json:"kgCommitted"`
	TypedIndexOutcome            enum.ComponentOutcome             `json:"typedIndexOutcome"`
	CrossDocumentIndexOutcome    enum.ComponentOutcome             `json:"crossDocumentIndexOutcome"`
	DerivedIndexOutcome          enum.DerivedIndexOutcome          `json:"derivedIndexOutcome"`
	ObservationProjectionOutcome enum.ObservationProjectionOutcome `json:"observationProjectionOutcome"`
	OuterTaskDisposition         enum.OuterTaskDisposition         `json:"outerTaskDisposition"`
	PythonInvocationOutcome      enum.InvocationOutcome            `json:"pythonInvocationOutcome"`
	AdvisorInvocationOutcome     enum.InvocationOutcome            `json:"advisorInvocationOutcome"`
	PythonExtractionStatus       string                            `json:"pythonExtractionStatus"`
	AdvisorReason                string                            `json:"advisorReason"`
	DegradationReasons           []string                          `json:"degradationReasons"`
	ExtractionMetadata           map[string]any                    `json:"extractionMetadata"`
	Attempt                      int                               `json:"attempt"`
	MaxAttempts                  int                               `json:"maxAttempts"`
}

// IsCommittedGraph 检查图谱是否已提交
func (p *GraphRagBuildResult) IsCommittedGraph() bool {
	return p != nil && p.KgCommitted && p.GraphPersistenceOutcome != "" &&
		p.OuterTaskDisposition != enum.OuterTaskDispositionFailIndexTask
}

// ResultAttempt 结果尝试次数
func (p *GraphRagBuildResult) ResultAttempt() int {
	if p == nil {
		return 0
	}
	return max(0, p.Attempt)
}

// ResultMaxAttempts 结果最大尝试次数
func (p *GraphRagBuildResult) ResultMaxAttempts() int {
	if p == nil || p.MaxAttempts == 0 {
		return max(1, p.ResultAttempt())
	}
	return max(1, p.MaxAttempts)
}

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
