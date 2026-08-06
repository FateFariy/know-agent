package analysis

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

// UploadPhase 上传阶段：上传解析后的纯文本到对象存储
type UploadPhase struct {
	port *adapter.DocumentPort
}

func NewUploadPhase(port *adapter.DocumentPort) *UploadPhase {
	return &UploadPhase{port: port}
}

func (p *UploadPhase) Name() string {
	return "上传阶段"
}

func (p *UploadPhase) Execute(ctx context.Context, parseCtx *Context) error {
	parsedTextPath, err := p.port.UploadParsedText(ctx, parseCtx.DocumentID, parseCtx.AnalysisResult.ParsedText)
	if err != nil {
		return err
	}
	parseCtx.ParsedTextPath = parsedTextPath
	return nil
}
