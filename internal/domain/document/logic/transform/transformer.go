package transform

import (
	"context"
)

type Transformer interface {
	Transform(ctx context.Context, text string, opts ...TransformerOption) (any, error)
}

type TransformerOption struct {
	implSpecificOptFn any
}

func WrapTransformerImplSpecificOptFn[T any](optFn func(*T)) TransformerOption {
	return TransformerOption{
		implSpecificOptFn: optFn,
	}
}

func GetTransformerImplSpecificOptions[T any](base *T, opts ...TransformerOption) *T {
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
