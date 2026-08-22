package analysis

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
)

// DownloadStage 下载阶段：从对象存储下载原始文件
type DownloadStage struct {
	storage adapter.Storage
}

func NewDownloadStage(storage adapter.Storage) *DownloadStage {
	return &DownloadStage{
		storage: storage,
	}
}

func (p *DownloadStage) Name() string {
	return "下载阶段"
}

func (p *DownloadStage) Execute(ctx context.Context, parseCtx *Context) error {
	rawFileBytes, err := p.storage.DownloadObject(ctx, parseCtx.Document.ObjectName)
	if err != nil {
		return err
	}
	parseCtx.RawFileBytes = rawFileBytes
	return nil
}
