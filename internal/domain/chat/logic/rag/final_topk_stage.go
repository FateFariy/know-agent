package rag

import (
	"context"
)

// FinalTopKStage 截取 topK 阶段，从重排序后的文档中截取 finalTopK 篇
type FinalTopKStage struct {
}

func NewFinalTopKStage() *FinalTopKStage {
	return &FinalTopKStage{}
}

func (s *FinalTopKStage) Name() string {
	return "FinalTopK"
}

// Execute 从重排序后的文档中截取前 finalTopK ，结果写入 state.FinalDocs
func (s *FinalTopKStage) Execute(_ context.Context, state *RetrievalState) error {
	if len(state.RerankedDocs) == 0 {
		return nil
	}
	finalTopK := min(state.Plan.FinalTopK, len(state.RerankedDocs))
	state.FinalDocs = state.RerankedDocs[:finalTopK]
	return nil
}
