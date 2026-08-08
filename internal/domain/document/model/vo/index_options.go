package vo

import "github.com/swiftbit/know-agent/internal/domain/document/model/enum"

type IndexingOptions struct {
	Chunk    ChunkOptions         `json:"chunk"`
	GraphRag GraphRagBuildOptions `json:"graphRag"`
	Raptor   RaptorBuildOptions   `json:"raptor"`
}

// ResolveRecursiveMaxChars 解析递归最大字符数
func (o *IndexingOptions) ResolveRecursiveMaxChars(pipelineType enum.PipelineType) int {
	if pipelineType == enum.PipelineTypeParent {
		return o.Chunk.ParentBlockMaxChars
	}
	return o.Chunk.ChildRecursiveMaxChars
}

// ResolveSemanticMaxChars 解析语义最大字符数
func (o *IndexingOptions) ResolveSemanticMaxChars(pipelineType enum.PipelineType) int {
	if pipelineType == enum.PipelineTypeParent {
		return o.Chunk.ParentSemanticMaxChars
	}
	return o.Chunk.ChildSemanticMaxChars
}

// ResolveSemanticMinChars 解析语义最小字符数
func (o *IndexingOptions) ResolveSemanticMinChars(pipelineType enum.PipelineType) int {
	if pipelineType == enum.PipelineTypeParent {
		return o.Chunk.ParentSemanticMinChars
	}
	return o.Chunk.ParentSemanticMinChars
}

// ResolveRecursiveOverlap 解析递归重叠字符数
func (o *IndexingOptions) ResolveRecursiveOverlap(pipelineType enum.PipelineType, maxChars int) int {
	if pipelineType == enum.PipelineTypeParent {
		return min(o.Chunk.ParentBlockOverlapChars, max(0, maxChars-1))
	}
	return min(o.Chunk.ChildRecursiveOverlapChars, max(0, maxChars-1))
}

// ResolveLlmMaxChars 解析LLM最大字符数
func (o *IndexingOptions) ResolveLlmMaxChars(pipelineType enum.PipelineType) int {
	if pipelineType == enum.PipelineTypeParent {
		return o.Chunk.ParentBlockMaxChars
	}
	return o.Chunk.ChildRecursiveMaxChars
}

type ChunkOptions struct {
	ChildRecursiveMaxChars           int     `json:"childRecursiveMaxChars"`
	ChildRecursiveOverlapChars       int     `json:"childRecursiveOverlapChars"`
	ChildSemanticMaxChars            int     `json:"childSemanticMaxChars"`
	ChildSemanticMinChars            int     `json:"childSemanticMinChars"`
	ChildSemanticSimilarityThreshold float64 `json:"childSemanticSimilarityThreshold"`
	ParentBlockMaxChars              int     `json:"parentBlockMaxChars"`
	ParentBlockOverlapChars          int     `json:"parentBlockOverlapChars"`
	ParentSemanticMaxChars           int     `json:"parentSemanticMaxChars"`
	ParentSemanticMinChars           int     `json:"parentSemanticMinChars"`
}

type GraphRagBuildOptions struct {
	GraphRagBuildEnabled bool `json:"graphRagBuildEnabled"`
}

type RaptorBuildOptions struct {
	RaptorBuildEnabled        bool    `json:"raptorBuildEnabled"`
	RaptorMaxClusterSize      int     `json:"raptorMaxClusterSize"`
	RaptorMaxLevels           int     `json:"raptorMaxLevels"`
	RaptorLlmSummaryEnabled   bool    `json:"raptorLlmSummaryEnabled"`
	RaptorSummaryQualityFloor float64 `json:"raptorSummaryQualityFloor"`
}
