package vo

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/shared"
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

func (c *ChunkCandidate) ExtractKeywords(tokenizer shared.Tokenizer) []string {
	if c == nil {
		return nil
	}
	seed := shared.NewKeywordSeed(c.Title, c.SectionPath, c.Text)
	return seed.Build(tokenizer)
}

func (c *ChunkCandidate) ExtractQuestions(keywords []string) []string {
	if c == nil {
		return nil
	}
	seed := shared.NewQuestionSeed(c.Title, c.ChunkType, keywords)
	return seed.Build()
}

func (c *ChunkCandidate) ExtractContentWithWeight(keywords []string, parserWeightedContent string) string {
	if c == nil {
		return ""
	}
	questions := c.ExtractQuestions(keywords)
	seed := shared.NewRichContentSeed(c.Text, c.SectionPath, c.Title, c.ChunkType, parserWeightedContent, keywords, questions)
	return seed.Build()
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

type DocumentStrategyStepDrafts []*DocumentStrategyStepDraft

func (d DocumentStrategyStepDrafts) PipelineSnapshot() string {
	if len(d) == 0 {
		return ""
	}
	steps := make([]string, 0, len(d))
	for _, step := range d {
		steps = append(steps, strconv.Itoa(step.StrategyType))
	}
	return strings.Join(steps, ",")
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

		candidate.Text = strutil.Trim(candidate.Text)
		if candidate.Text != "" {
			path := utils.BlankToDefault(candidate.CanonicalPath, candidate.SectionPath)
			uniqueKey := fmt.Sprintf("%s||%d||%s", path, candidate.ItemIndex, candidate.Text)
			if _, ok := seen[uniqueKey]; !ok {
				seen[uniqueKey] = struct{}{}
				result = append(result, candidate)
			}
		}
	}

	return result
}

type ParentChunkCandidates []*ParentChunkCandidate

// CleanupAndUnique 过滤空文本并按 路径+序号+文本 去重
func (p ParentChunkCandidates) CleanupAndUnique() ParentChunkCandidates {
	if len(p) == 0 {
		return p
	}
	seen := make(map[string]struct{})
	result := make(ParentChunkCandidates, 0, len(p))
	for _, candidate := range p {
		if candidate == nil {
			continue
		}

		candidate.Text = strutil.Trim(candidate.Text)
		if candidate.Text != "" {
			path := utils.BlankToDefault(candidate.CanonicalPath, candidate.SectionPath)
			uniqueKey := fmt.Sprintf("%s||%d||%s", path, candidate.ItemIndex, candidate.Text)
			if _, ok := seen[uniqueKey]; !ok {
				seen[uniqueKey] = struct{}{}
				result = append(result, candidate)
			}
		}
	}

	return result
}

// CleanupParentCandidates 过滤"文本为空"或"无子块"的父块候选
func (p ParentChunkCandidates) CleanupParentCandidates() ParentChunkCandidates {
	candidates := make(ParentChunkCandidates, 0, len(p))
	fn := func(child *ChunkCandidate) bool {
		return child != nil && strutil.IsNotBlank(child.Text)
	}
	for _, candidate := range p {
		if candidate == nil || len(candidate.ChildChunks) == 0 ||
			strutil.IsBlank(candidate.Text) || slices.ContainsFunc(candidate.ChildChunks, fn) {
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates
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
