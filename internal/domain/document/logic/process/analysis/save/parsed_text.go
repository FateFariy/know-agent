package save

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

type ParsedTextUploadPhase struct {
	port *adapter.DocumentPort
}

func NewParsedTextUploadPhase(port *adapter.DocumentPort) *ParsedTextUploadPhase {
	return &ParsedTextUploadPhase{
		port: port,
	}
}

func (p *ParsedTextUploadPhase) Name() string {
	return "解析文本上传阶段"
}

func (p *ParsedTextUploadPhase) Execute(ctx context.Context, saveCtx *Context) error {
	parsedTextPath, err := p.port.UploadParsedText(ctx, saveCtx.DocumentId, saveCtx.AnalysisResult.ParsedText)
	if err != nil {
		return err
	}
	saveCtx.ParsedTextPath = parsedTextPath
	return nil
}
