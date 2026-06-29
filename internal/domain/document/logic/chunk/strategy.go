package chunk

import (
	"context"
)

// TextBlock 文本块通用实体
type TextBlock struct {
	SectionPath   string // 文本所属章节路径，例如 "1.1.2"
	CanonicalPath string // 文本唯一标准层级路径，例如 "1.1.2.1"
	ItemIndex     int    // 原始文档片段在来源列表中的下标索引
	Text          string // 原始/切分后文本内容
	SourceType    int    // 来源类型
}

func (t *TextBlock) CloneWithText(text string) *TextBlock {
	return &TextBlock{
		SectionPath:   t.SectionPath,
		CanonicalPath: t.CanonicalPath,
		ItemIndex:     t.ItemIndex,
		Text:          text,
		SourceType:    t.SourceType,
	}
}

// Strategy 分块策略接口
type Strategy interface {
	// Name 策略名称，唯一标识策略
	Name() string

	// Chunk 将一段输入文本切分为多个文本块
	Chunk(ctx context.Context, input *TextBlock, opts ...Option) ([]*TextBlock, error)
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
