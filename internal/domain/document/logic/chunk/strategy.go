package chunk

import (
	"context"
)

// PipelineType 流水线类型：PARENT 表示父块流水线，CHILD 表示子块流水线
// 在不同流水线中同一策略的参数（如最大字符数）通常不同，由策略实现自行区分
type PipelineType = string

const (
	// PipelineTypeParent 父块流水线
	PipelineTypeParent PipelineType = "PARENT"

	// PipelineTypeChild 子块流水线
	PipelineTypeChild PipelineType = "CHILD"
)

// Input 分块策略的输入对象
// 封装单个文本片段及其来源信息，便于各策略复用与扩展
type Input struct {
	// SectionPath 文本所属章节路径（可选）
	SectionPath string

	// CanonicalPath 文本的标准路径，用于去重和溯源（可选）
	CanonicalPath string

	// ItemIndex 文本在来源列表中的索引，用于去重和保持顺序
	ItemIndex int

	// Text 输入原文，不能为空或空白
	Text string

	// SourceType 来源类型，由外部语义控制（例如结构节点、解析文本等）
	SourceType int
}

// Output 分块策略的输出对象
// 表示切分出来的一个文本块
type Output struct {
	// SectionPath 文本所属章节路径
	SectionPath string

	// CanonicalPath 文本的标准路径
	CanonicalPath string

	// ItemIndex 对应来源索引
	ItemIndex int

	// Text 切分后的文本
	Text string

	// SourceType 来源类型
	SourceType int
}

// Strategy 分块策略接口
// 每种具体的分块策略（结构 / 递归 / 语义 / 大模型）都应实现该接口
// 通过统一的 Chunk 方法暴露，使上层可以无差别地对任意策略进行调用和编排
type Strategy interface {
	// Name 策略名称，唯一标识策略，用于日志、注册、调试
	Name() string

	// Chunk 将一段输入文本切分为多个文本块
	// ctx 用于传递请求链路信息、超时控制等
	// input 输入文本以及它的元数据
	// pipelineType 标识该调用发生在哪一条流水线（PARENT / CHILD），
	//   不同流水线通常要求不同的块大小/重叠等参数
	// 返回切分后的块列表，空文本或无法切分的输入应返回空切片
	Chunk(ctx context.Context, input *Input, pipelineType PipelineType) ([]*Output, error)
}

// StrategyOption 用于配置策略的函数选项
type StrategyOption func(*baseOptions)

// baseOptions 策略通用配置项，对外部不可见
// 仅通过 Option 函数进行构造，避免字段扩散
type baseOptions struct {
	recursiveMaxChars           int
	recursiveOverlapChars       int
	semanticMaxChars            int
	semanticMinChars            int
	semanticSimilarityThreshold float64
	llmEnabled                  bool
	llmMaxChars                 int
}

// WithRecursive 设置递归切块相关参数
// maxChars 单个块的最大字符数；overlapChars 相邻块的重叠字符数
func WithRecursive(maxChars, overlapChars int) StrategyOption {
	return func(o *baseOptions) {
		if maxChars > 0 {
			o.recursiveMaxChars = maxChars
		}
		if overlapChars >= 0 {
			o.recursiveOverlapChars = overlapChars
		}
	}
}

// WithSemantic 设置语义切块相关参数
// minChars 触发语义切分的最小字符数；maxChars 单个块的最大字符数
// threshold Jaccard 相似度阈值，小于该阈值则触发语义切分
func WithSemantic(minChars, maxChars int, threshold float64) StrategyOption {
	return func(o *baseOptions) {
		if minChars > 0 {
			o.semanticMinChars = minChars
		}
		if maxChars > 0 {
			o.semanticMaxChars = maxChars
		}
		if threshold > 0 && threshold <= 1 {
			o.semanticSimilarityThreshold = threshold
		}
	}
}

// WithLLM 配置大模型切块的相关参数
// enabled 是否启用；maxChars 单次调用大模型允许的最大字符数
func WithLLM(enabled bool, maxChars int) StrategyOption {
	return func(o *baseOptions) {
		o.llmEnabled = enabled
		if maxChars > 0 {
			o.llmMaxChars = maxChars
		}
	}
}

// applyOptions 应用选项，返回一组默认值与用户设置合并后的配置
func applyOptions(opts []StrategyOption) baseOptions {
	base := baseOptions{
		recursiveMaxChars:           800,
		recursiveOverlapChars:       120,
		semanticMaxChars:            700,
		semanticMinChars:            240,
		semanticSimilarityThreshold: 0.18,
		llmEnabled:                  false,
		llmMaxChars:                 3500,
	}
	for _, opt := range opts {
		opt(&base)
	}
	return base
}
