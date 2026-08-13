package rank

import (
	"context"
	"fmt"

	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

type DocumentRanker struct {
	repo                   adapter.KnowledgeRepository
	docGateway             adapter.DocumentGateway
	lowConfidenceThreshold float64
	documentCandidateLimit int
}

func (r *DocumentRanker) Rank(ctx context.Context, rankCtx *Context) error {
	documents, err := r.docGateway.FindDocumentProfileByDocIds(ctx)
	if err != nil {
		return fmt.Errorf("查询可检索文档失败: %w", err)
	}
	if len(documents) == 0 {
		rankCtx.DocumentCandidates = []*vo.DocumentRouteCandidate{}
		return nil
	}

	// 加载画像、关系等（略）
	// 获取 top scope/topic 从 q 中读取（因为顺序执行时已填充）
	topScopeCode := ""
	if len(rankCtx.ScopeCandidates) > 0 {
		topScopeCode = rankCtx.ScopeCandidates[0].ScopeCode
	}
	topTopicCode := ""
	if len(rankCtx.TopicCandidates) > 0 {
		topTopicCode = rankCtx.TopicCandidates[0].TopicCode
	}

	// 计算语义分、词面分等（略）
	// 打分逻辑（参照原 rankDocuments）
	// 最终赋值 q.DocumentCandidates
	return nil
}
