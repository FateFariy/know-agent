package storage

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/rank"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

type ElasticStorage struct {
}

func NewElasticStorage(svcCtx *svc.ServiceContext) *ElasticStorage {
	return &ElasticStorage{}
}

func (e *ElasticStorage) Search(ctx context.Context, input *rank.EsSearchInput) ([]*vo.RouteLexicalHit, error) {
	// 实现逻辑
	return nil, nil
}
