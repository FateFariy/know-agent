package analysis

import (
	"context"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
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
	downloadStartedNanos := time.Now()

	rawFileBytes, err := p.port.DownloadObject(ctx, parseCtx.Document.ObjectName)
	if err != nil {
		return err
	}
	parseCtx.RawFileBytes = rawFileBytes
	downloadCostMillis := time.Since(downloadStartedNanos).Milliseconds()

	logx.Infof("解析源文件下载完成，documentId=%d, taskId=%d, objectName=%s, fileSizeBytes=%d, costMillis=%d",
		parseCtx.DocumentID, parseCtx.TaskID, parseCtx.Document.ObjectName, len(rawFileBytes), downloadCostMillis)
	return nil
}
