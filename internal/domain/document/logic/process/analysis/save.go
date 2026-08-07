package analysis

import (
	"context"
	"fmt"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/analysis/save"
)

// Save 保存阶段子任务接口
type Save interface {
	Name() string
	Execute(ctx context.Context, saveCtx *save.Context) error
}

// SavePhase 保存阶段：按顺序执行文本上传、产物持久化、结构节点持久化等子阶段
type SavePhase struct {
	phases []Save
}

// NewSavePhase 创建保存阶段
func NewSavePhase(repo adapter.DocumentRepository, tableRepo adapter.TableRepository, port *adapter.DocumentPort) *SavePhase {
	return &SavePhase{
		phases: []Save{
			save.NewParsedTextUploadPhase(port),
			save.NewArtifactPersistPhase(repo, tableRepo, port),
			save.NewStructurePersistPhase(repo),
			save.NewNavigationUploadPhase(port),
			save.NewProfileGeneratePhase(),
		},
	}
}

// Name 阶段名称
func (p *SavePhase) Name() string {
	return "保存阶段"
}

// Execute 执行保存阶段，从解析上下文构建保存上下文，依次执行各子阶段
func (p *SavePhase) Execute(ctx context.Context, parseCtx *Context) error {
	// 构建保存上下文
	saveCtx := &save.Context{
		DocumentId:     parseCtx.DocumentId,
		TaskId:         parseCtx.TaskId,
		AnalysisResult: parseCtx.AnalysisResult,
	}

	// 依次执行各子阶段
	for _, stage := range p.phases {
		stageName := stage.Name()
		startTime := time.Now()

		logx.Infof("[SavePhase] 开始执行子阶段: %s, documentId=%d, taskId=%d", stageName, parseCtx.DocumentId, parseCtx.TaskId)

		if err := stage.Execute(ctx, saveCtx); err != nil {
			return fmt.Errorf("保存子阶段 %s 执行失败: %w", stageName, err)
		}

		logx.Infof("[SavePhase] 子阶段 %s 执行成功, costMillis=%d", stageName, time.Since(startTime).Milliseconds())
	}

	parseCtx.SaveCtx = saveCtx

	return nil
}
