package analysis

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

// DownloadStage 下载阶段：从对象存储下载原始文件
type DownloadStage struct {
	port *adapter.DocumentPort
}

func NewDownloadStage(port *adapter.DocumentPort) *DownloadStage {
	return &DownloadStage{port: port}
}

func (p *DownloadStage) Name() string {
	return "下载阶段"
}

func (p *DownloadStage) Execute(ctx context.Context, parseCtx *Context) error {
	rawFileBytes, err := p.port.DownloadObject(ctx, parseCtx.Document.ObjectName)
	if err != nil {
		return err
	}
	parseCtx.RawFileBytes = rawFileBytes
	return nil
}
