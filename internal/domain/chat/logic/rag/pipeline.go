package rag

import (
	"context"
	"fmt"
	"time"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// RetrievalState 承载单个子问题检索管线的所有输入、中间产物和最终结果
type RetrievalState struct {
	// 输入
	Input           *ExecutionInput
	RetrievalResult *vo.RetrievalResult
	Plan            *vo.RetrievalPlan
	Start           time.Time

	// 阶段中间产物
	ChannelResults         []*RetrievalChannelResult
	ChannelTraces          []*vo.SubQuestionChannelTrace
	FusedDocs              []*vo.DocumentChunk
	ParentSearchDocs       []*vo.DocumentChunk
	RerankedDocs           []*vo.DocumentChunk
	FinalDocs              []*vo.DocumentChunk
	ObservationPersistence *vo.ObservationPersistence
}

// Stage 检索管线阶段接口
type Stage interface {
	Name() string

	Execute(ctx context.Context, state *RetrievalState) error
}

// Pipeline 按顺序执行一组阶段的管线执行器。
type Pipeline struct {
	stages []Stage
}

// NewPipeline 创建管线，按传入顺序依次执行阶段。
func NewPipeline(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

// Execute 按顺序执行所有阶段，遇到错误立即终止，返回包装后的错误。
func (p *Pipeline) Execute(ctx context.Context, state *RetrievalState) error {
	for _, stage := range p.stages {
		// 提前检查 ctx 是否已取消
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := stage.Execute(ctx, state); err != nil {
			return fmt.Errorf("stage %s failed: %w", stage.Name(), err)
		}
	}
	return nil
}
