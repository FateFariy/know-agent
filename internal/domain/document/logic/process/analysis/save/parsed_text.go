package save

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

type ParsedTextUploadPhase struct {
	storage adapter.Storage
}

func NewParsedTextUploadPhase(storage adapter.Storage) *ParsedTextUploadPhase {
	return &ParsedTextUploadPhase{
		storage: storage,
	}
}

func (p *ParsedTextUploadPhase) Name() string {
	return "解析文本上传阶段"
}

func (p *ParsedTextUploadPhase) Execute(ctx context.Context, saveCtx *Context) error {
	parsedTextPath, err := p.storage.UploadParsedText(ctx, saveCtx.DocumentId, saveCtx.AnalysisResult.ParsedText)
	if err != nil {
		return err
	}
	saveCtx.ParsedTextPath = parsedTextPath
	return nil
}
