package analysis

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
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
	parsedTextUploadStartTime := time.Now()

	parsedTextPath, err := p.port.UploadParsedText(ctx, parseCtx.DocumentID, parseCtx.AnalysisResult.ParsedText)
	if err != nil {
		return err
	}
	parseCtx.ParsedTextPath = parsedTextPath
	costMillis := time.Since(parsedTextUploadStartTime).Milliseconds()

	logx.Infof("解析文本上传完成，documentId=%d, taskId=%d, parseTextPath=%s, charCount=%d, costMillis=%d",
		parseCtx.DocumentID, parseCtx.TaskID, parsedTextPath, parseCtx.AnalysisResult.CharCount, costMillis)
	return nil
}
