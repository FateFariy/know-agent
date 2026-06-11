package node

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/load"
)

type LoaderNode struct {
	factory *load.Factory
}

var _ document.Loader = (*LoaderNode)(nil)

func NewLoaderNode(factory *load.Factory) *LoaderNode {
	return &LoaderNode{
		factory: factory,
	}
}

// Load 加载文档
func (l *LoaderNode) Load(ctx context.Context, src document.Source, opts ...document.LoaderOption) ([]*schema.Document, error) {
	var err error

	// 1. 处理错误，并进行错误回调方法
	defer func() {
		if err != nil {
			callbacks.OnError(ctx, err)
		}
	}()

	// 2. 开始加载前的回调
	ctx = callbacks.OnStart(ctx, &document.LoaderCallbackInput{
		Source: src,
	})

	// 3. 执行加载逻辑
	docs, err := l.doLoad(ctx, src)

	if err != nil {
		return nil, err
	}

	ctx = callbacks.OnEnd(ctx, &document.LoaderCallbackOutput{
		Source: src,
		Docs:   docs,
	})

	return docs, nil
}

// doLoad 执行加载逻辑
func (l *LoaderNode) doLoad(ctx context.Context, src document.Source) ([]*schema.Document, error) {
	// 1. 获取 Loader 实现
	// todo 暂时只支持本地文件加载
	loader, err := l.factory.GetLoader(load.TypeFile)
	if err != nil {
		return nil, err
	}

	// 2. 加载文档内容
	docs, err := loader.Load(ctx, src)
	if err != nil {
		return nil, err
	}

	return docs, nil
}
