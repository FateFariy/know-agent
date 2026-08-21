package rag

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/rerank"
)

// RerankStage 重排序阶段，对父块提升后的文档执行重排序（如果启用）
type RerankStage struct {
	reranker rerank.Reranker
	enabled  bool
}

// NewRerankStage 创建重排序阶段
func NewRerankStage(reranker rerank.Reranker, enabled bool) *RerankStage {
	return &RerankStage{
		reranker: reranker,
		enabled:  enabled,
	}
}

func (s *RerankStage) Name() string {
	return "Rerank"
}

// Execute 对父块提升后的文档执行重排序，结果写入 state.RerankedDocs。
func (s *RerankStage) Execute(ctx context.Context, state *RetrievalState) error {
	state.RerankedDocs = state.ParentSearchDocs
	if !s.enabled || len(state.RerankedDocs) == 0 || s.reranker == nil {
		return nil
	}

	result, err := s.reranker.Process(ctx, state.Input.SubQuestion, state.ParentSearchDocs)
	if err != nil {
		logx.Warnf("重排序处理失败: subQuestion='%s', error=%v", state.Input.SubQuestion, err)
		return err
	}
	state.RerankedDocs = result

	return nil
}
