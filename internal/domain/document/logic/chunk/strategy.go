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

// Option 用于配置策略的函数选项。
// 各子包（recursive/semantic/llm/structure）通过 WrapChunkImplSpecificOptFn 构造
// 提供独立的参数配置。
type Option struct {
	implSpecificOptFn any
}

// WrapChunkImplSpecificOptFn 将一个策略专属的 option setter 函数封装为通用 Option。
// 传入的闭包将在 GetChunkImplSpecificOptions 中被调用以填充对应策略的 options。
func WrapChunkImplSpecificOptFn[T any](optFn func(*T)) Option {
	return Option{
		implSpecificOptFn: optFn,
	}
}

// GetChunkImplSpecificOptions 从一组通用 Option 中筛选出 T 类型的 setter，
// 并应用到传入的 base 上。当 base 为 nil 时会新建一个 T。
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
