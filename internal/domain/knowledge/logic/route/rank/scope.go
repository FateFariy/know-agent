package rank

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/score"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

type ScopeRanker struct {
	repo adapter.KnowledgeRepository
	*base
}

func NewScopeRanker(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway, opts ...Option) *ScopeRanker {
	return &ScopeRanker{
		repo: repo,
		base: newBaseRanker(docGateway, opts...),
	}
}

func (r *ScopeRanker) Rank(ctx context.Context, rankCtx *Context) error {
	nodes, err := r.repo.SelectKnowledgeScopeNodesByKbIds(ctx, rankCtx.SelectedKnowledgeBaseIds)
	if err != nil {
		return fmt.Errorf("查询 scope 节点失败: %w", err)
	}
	if len(nodes) == 0 {
		return nil
	}

	routeTexts := utils.Map(nodes, func(node *entity.KnowledgeScopeNode) string {
		return utils.JoinNonBlank(" ", node.ScopeName, node.Description, node.Aliases, node.Examples)
	})
	semanticScores := r.computeSemanticScores(ctx, rankCtx, routeTexts)
	lexicalScores := r.searchLexicalScores(ctx, rankCtx, "scope", 5)

	candidates := make([]*vo.ScopeRouteCandidate, 0, len(nodes))
	for i, node := range nodes {
		scoreResult := r.scorer.Score(&score.Features{
			SemanticScore: semanticScores[i],
			LexicalScore:  lexicalScores[node.ID],
		})
		if scoreResult.TotalScore > 0 {
			candidates = append(candidates, &vo.ScopeRouteCandidate{
				ScopeId:   node.ID,
				ScopeName: node.ScopeName,
				Score:     scoreResult.TotalScore,
				Reason:    scoreResult.Reason,
				Source:    scoreResult.Source,
				Features:  scoreResult.Features,
			})
		}
	}
	slices.SortFunc(candidates, func(a, b *vo.ScopeRouteCandidate) int { return -cmp.Compare(a.Score, b.Score) })
	rankCtx.ScopeCandidates = utils.Limit(candidates, 5)
	return nil
}
