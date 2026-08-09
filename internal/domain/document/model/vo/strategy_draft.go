package vo

import (
	"fmt"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
)

// DocumentStrategyPlanDraft 策略方案草稿
type DocumentStrategyPlanDraft struct {
	ParentSteps      []*DocumentStrategyStepDraft // 父块步骤列表
	ChildSteps       []*DocumentStrategyStepDraft // 子块步骤列表
	StrategySnapshot string                       // 策略快照
	RecommendReason  string                       // 推荐理由
}

// DocumentStrategyStepDraft 策略步骤草稿
type DocumentStrategyStepDraft struct {
	PipelineType    string // 流水线类型
	StrategyType    int    // 策略类型
	StrategyRole    int    // 策略角色
	SourceType      int    // 来源类型
	RecommendReason string // 推荐理由
}

// ChunkCandidate 块候选
type ChunkCandidate struct {
	SectionPath       string // 章节路径
	StructureNodeId   int64  // 结构体节点ID
	StructureNodeType int    // 结构体节点类型
	CanonicalPath     string // 标准路径
	ItemIndex         int    // 项目索引
	Text              string // 文本内容
	SourceType        int    // 来源类型
	ContentWithWeight string // 带权重的内容
	ChunkType         string // 块类型
	Title             string // 标题
	Keywords          string // 关键词
	Questions         string // 问题
	PageNo            int    // 页码
	PageRange         string // 页码范围
	BboxJson          string // 边界框JSON
	SourceBlockIds    string // 源块ID列表
}

// ParentChunkCandidate 父块候选
type ParentChunkCandidate struct {
	SectionPath       string            // 章节路径
	StructureNodeId   int64             // 结构体节点ID
	StructureNodeType int               // 结构体节点类型
	CanonicalPath     string            // 标准路径
	ItemIndex         int               // 项目索引
	Text              string            // 文本内容
	SourceType        int               // 来源类型
	PageRange         string            // 页码范围
	SourceBlockIds    string            // 源ID列表
	ChildChunks       []*ChunkCandidate // 子块列表
}

type ParentChunkCandidates []*ParentChunkCandidate

// CleanupAndUnique 过滤空文本并按 路径+序号+文本 去重
func (p ParentChunkCandidates) CleanupAndUnique() ParentChunkCandidates {
	seen := make(map[string]struct{})
	result := make(ParentChunkCandidates, 0, len(p))
	for _, candidate := range p {
		if candidate == nil {
			continue
		}

		trim := strutil.Trim(candidate.Text)
		if trim != "" {
			path := utils.BlankToDefault(candidate.CanonicalPath, candidate.SectionPath)
			uniqueKey := fmt.Sprintf("%s||%d||%s", path, candidate.ItemIndex, trim)
			if _, ok := seen[uniqueKey]; !ok {
				seen[uniqueKey] = struct{}{}
				result = append(result, candidate)
			}
		}
	}

	return result
}

func (p ParentChunkCandidates) CountChildCandidates() int {
	count := 0
	for _, candidate := range p {
		for _, child := range candidate.ChildChunks {
			if child != nil && strutil.IsNotBlank(child.Text) {
				count++
			}
		}
	}
	return count
}

type ChunkCandidates []*ChunkCandidate

// CleanupAndUnique 过滤空文本并按 路径+序号+文本 去重
func (c ChunkCandidates) CleanupAndUnique() ChunkCandidates {
	seen := make(map[string]struct{})
	result := make(ChunkCandidates, 0, len(c))
	for _, candidate := range c {
		if candidate == nil {
			continue
		}

		trim := strutil.Trim(candidate.Text)
		if trim != "" {
			path := utils.BlankToDefault(candidate.CanonicalPath, candidate.SectionPath)
			uniqueKey := fmt.Sprintf("%s||%d||%s", path, candidate.ItemIndex, trim)
			if _, ok := seen[uniqueKey]; !ok {
				seen[uniqueKey] = struct{}{}
				result = append(result, candidate)
			}
		}
	}

	return result
}
