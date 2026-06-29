package chunk

import (
	"context"
)

// Input 分块策略的输入对象，封装单个文本片段及其来源信息
type Input struct {
	SectionPath   string // 文本所属章节路径，例如 "1.1.2"
	CanonicalPath string // 文本的标准路径，例如 "1.1.2.1"
	ItemIndex     int    // 文本在来源列表中的索引，例如 0
	Text          string // 文本内容
	SourceType    int    // 来源类型，例如 1 表示结构节点，2 表示解析文本等
}

// Output 分块策略的输出对象，表示切分出来的一个文本块
type Output struct {
	SectionPath   string // 文本所属章节路径，例如 "1.1.2"
	CanonicalPath string // 文本的标准路径，例如 "1.1.2.1"
	ItemIndex     int    // 文本在来源列表中的索引，例如 0
	Text          string // 切切分后的文本内容
	SourceType    int    // 来源类型，例如 1 表示结构节点，2 表示解析文本等
}

// Strategy 分块策略接口
// 每种具体的分块策略都实现该接口。策略本身不感知调用方的流水线类型，
// 块大小/重叠等参数由调用方通过 Option 函数显式传入
type Strategy interface {
	// Name 策略名称，唯一标识策略，用于日志、注册、调试
	Name() string

	// Chunk 将一段输入文本切分为多个文本块
	Chunk(ctx context.Context, input *Input, opts ...Option) ([]*Output, error)
}

// Option 配置策略的函数选项
type Option struct {
	implSpecificOptFn any
}

// WrapChunkImplSpecificOptFn 将策略专属的 option 函数封装为通用 Option
func WrapChunkImplSpecificOptFn[T any](optFn func(*T)) Option {
	return Option{
		implSpecificOptFn: optFn,
	}
}

// GetChunkImplSpecificOptions 从 Option 列表中获取策略实现专有选项
func GetChunkImplSpecificOptions[T any](base *T, opts ...Option) *T {
	if base == nil {
		base = new(T)
	}

	for i := range opts {
		opt := opts[i]
		if opt.implSpecificOptFn != nil {
			s, ok := opt.implSpecificOptFn.(func(*T))
			if ok {
				s(base)
			}
		}
	}

	return base
}
