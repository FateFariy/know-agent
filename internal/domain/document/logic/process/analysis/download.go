package analysis

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

// DownloadPhase 下载阶段：从对象存储下载原始文件
type DownloadPhase struct {
	port *adapter.DocumentPort
}

func NewDownloadPhase(port *adapter.DocumentPort) *DownloadPhase {
	return &DownloadPhase{port: port}
}

func (p *DownloadPhase) Name() string {
	return "下载阶段"
}

func (p *DownloadPhase) Execute(ctx context.Context, parseCtx *Context) error {
	rawFileBytes, err := p.port.DownloadObject(ctx, parseCtx.Document.ObjectName)
	if err != nil {
		return err
	}
	parseCtx.RawFileBytes = rawFileBytes
	return nil
}
