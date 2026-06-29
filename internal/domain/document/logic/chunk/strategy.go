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
	SourceType    int    // 来源类型
}

// Output 分块策略的输出对象，表示切分出来的一个文本块
type Output struct {
	SectionPath   string // 文本所属章节路径，例如 "1.1.2"
	CanonicalPath string // 文本的标准路径，例如 "1.1.2.1"
	ItemIndex     int    // 文本在来源列表中的索引，例如 0
	Text          string // 切切分后的文本内容
	SourceType    int    // 来源类型
}

// Strategy 分块策略接口
type Strategy interface {
	// Name 策略名称，唯一标识策略
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
