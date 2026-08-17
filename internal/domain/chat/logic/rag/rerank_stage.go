package rag

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/rerank"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
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
	state.RerankedDocs = s.applyRerank(ctx, state.RetrievalResult, state.ParentSearchDocs, state.Input.ExecutionQuery)
	return nil
}

// applyRerank 应用重排序（如果启用）
func (s *RerankStage) applyRerank(ctx context.Context, retrievalResult *vo.RetrievalResult, candidates []*vo.DocumentChunk, subQuestion string) []*vo.DocumentChunk {
	if !s.enabled || len(candidates) == 0 || s.reranker == nil {
		return candidates
	}

	result, err := s.reranker.Process(ctx, subQuestion, candidates)
	if err != nil {
		logx.Warnf("重排序处理失败: subQuestion='%s', error=%v", subQuestion, err)
		return candidates
	}
	return result
}
