package chunk

import (
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/document/adapter/model"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/llm"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/recursive"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/semantic"
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
	"github.com/swiftbit/know-agent/internal/svc"
)

type Registry struct {
	registry map[int]Chunker
}

func NewChunkStrategyRegistry(svcCtx *svc.ServiceContext, chatModel model.ChatModel, template adapter.PromptRenderer) *Registry {
	registry := make(map[int]Chunker)
	// 递归分块
	registry[enum.StrategyTypeRecursive] = recursive.NewChunker(
		recursive.WithMaxChars(svcCtx.Config.Chunk.RecursiveMaxChars),
		recursive.WithOverlapChars(svcCtx.Config.Chunk.RecursiveOverlapChars),
	)

	// 语义分块
	registry[enum.StrategyTypeSemantic] = semantic.NewChunker(
		semantic.WithMinChars(svcCtx.Config.Chunk.SemanticMinChars),
		semantic.WithMaxChars(svcCtx.Config.Chunk.SemanticMaxChars),
		semantic.WithSimilarityThreshold(svcCtx.Config.Chunk.SemanticSimilarityThreshold),
	)
	// 大模型切块
	registry[enum.StrategyTypeLLM] = llm.NewChunker(chatModel, template,
		llm.WithLlmSplitPrompt(enum.DocumentLlmSplit),
	)

	return &Registry{registry: registry}
}

func (r *Registry) Get(strategy int) Chunker {
	return r.registry[strategy]
}
