package load

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino-ext/components/document/loader/url"
	"github.com/cloudwego/eino-ext/components/document/parser/pdf"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/document/parser"
)

const (
	TypeFile = "file"
	TypeUrl  = "url"
)

type Factory struct {
	loaders map[string]document.Loader
}

func NewFactory() *Factory {
	f := &Factory{
		loaders: make(map[string]document.Loader),
	}
	f.addFileLoader()
	f.addUrlLoader()
	return f
}

func (f *Factory) GetLoader(loaderType string) (document.Loader, error) {
	if _, ok := f.loaders[loaderType]; !ok {
		return nil, fmt.Errorf("不支持的来源类型 %s", loaderType)
	}
	return f.loaders[loaderType], nil
}

// addFileLoader 向工厂添加一个文件加载器
func (f *Factory) addFileLoader() {
	ctx := context.Background()
	pdfParser, _ := pdf.NewPDFParser(ctx, &pdf.Config{})
	extParser, _ := parser.NewExtParser(ctx, &parser.ExtParserConfig{
		Parsers: map[string]parser.Parser{
			".pdf": pdfParser,
		},
		FallbackParser: parser.TextParser{},
	})
	fileLoader, _ := file.NewFileLoader(ctx, &file.FileLoaderConfig{
		UseNameAsID: true,
		Parser:      extParser,
	})
	f.loaders[TypeFile] = fileLoader
}

// addUrlLoader 向工厂添加一个URL加载器
func (f *Factory) addUrlLoader() {
	loader, _ := url.NewLoader(context.Background(), &url.LoaderConfig{})
	f.loaders[TypeUrl] = loader
}
