package logic

import "github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"

type KnowledgeConfigLogicImpl struct {
	repo adapter.KnowledgeRepository
}

func NewKnowledgeConfigLogicImpl(repo adapter.KnowledgeRepository) *KnowledgeConfigLogicImpl {
	return &KnowledgeConfigLogicImpl{
		repo: repo,
	}
}
