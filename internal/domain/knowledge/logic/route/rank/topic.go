package rank

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/score"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

const (
	candidatesLimit = 8
)

type TopicRanker struct {
	repo   adapter.KnowledgeRepository
	scorer score.Scorer
	base
}

func NewTopicRanker(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway, embedder adapter.Embedder,
	lexicalIndex adapter.RouteLexicalIndex, scorer score.Scorer) *TopicRanker {
	return &TopicRanker{
		repo:   repo,
		scorer: scorer,
		base:   newBaseRanker(docGateway, embedder, lexicalIndex),
	}
}

func (r *TopicRanker) Rank(ctx context.Context, rankCtx *Context) error {
	nodes, err := r.repo.SelectKnowledgeTopicNodesByKbIds(ctx, nil)
	if err != nil {
		return fmt.Errorf("查询 topic 节点失败: %w", err)
	}
	if len(nodes) == 0 {
		return nil
	}

	preferredScopes := utils.MapBy(rankCtx.ScopeCandidates, func(candidate *vo.ScopeRouteCandidate) (int64, bool) {
		return candidate.ScopeId, true
	})

	routeTexts := utils.Map(nodes, func(node *entity.KnowledgeTopicNode) string {
		return utils.JoinNonBlank(" ", node.TopicName, node.Description, node.Aliases, node.Examples, node.AnswerShape, node.ExecutionPreference)
	})

	semanticScores := r.computeSemanticScores(ctx, rankCtx, routeTexts)
	lexicalScores := r.searchLexicalScores(ctx, rankCtx, "topic", 8)

	candidates := make([]*vo.TopicRouteCandidate, 0, len(nodes))
	for i, node := range nodes {
		scoreResult := r.scorer.Score(&score.Features{
			SemanticScore: semanticScores[i],
			LexicalScore:  lexicalScores[node.ID],
			RelationScore: utils.Ternary(preferredScopes[node.ScopeId], 1.0, 0.0),
		})
		if scoreResult.TotalScore > 0 {
			candidates = append(candidates, &vo.TopicRouteCandidate{
				TopicId:   node.ID,
				TopicName: node.TopicName,
				Score:     scoreResult.TotalScore,
				Reason:    scoreResult.Reason,
				Source:    scoreResult.Source,
				Features:  scoreResult.Features,
			})
		}
	}
	slices.SortFunc(candidates, func(a, b *vo.TopicRouteCandidate) int { return -cmp.Compare(a.Score, b.Score) })
	rankCtx.TopicCandidates = candidates
	return nil
}

// deriveTopicsFromProfiles 当 topic 节点未配置时，按文档画像的 CoreTopics 派生主题候选
func (r *TopicRanker) deriveTopicsFromProfiles(ctx context.Context, rankCtx *Context) ([]*vo.TopicRouteCandidate, error) {
	docs, _ := r.listRetrievableDocuments(ctx, rankCtx)
	if len(docs) == 0 {
		return nil, nil
	}
	slices.SortFunc(docs, func(a, b *vo.DocumentMetadata) int { return cmp.Compare(a.DocumentId, b.DocumentId) })

	docIds := utils.Map(docs, func(d *vo.DocumentMetadata) int64 { return d.DocumentId })
	profiles, err := r.docGateway.FindDocumentProfileByDocIds(ctx, docIds)
	if err != nil {
		return nil, fmt.Errorf("查询文档画像失败: %w", err)
	}

	best := make(map[string]*vo.TopicRouteCandidate)
	for _, profile := range profiles {
		for _, topic := range parseJsonStringArray(profile.CoreTopics) {
			routeText := utils.JoinNonBlank(" ", topic, profile.DocumentSummary, profile.ExampleQuestions)
			semanticScore := r.computeSemanticScores(ctx, rankCtx, []string{routeText})[0]
			scoreResult := r.scorer.Score(&score.Features{SemanticScore: semanticScore})
			if existing, ok := best[topic]; !ok || existing.Score < scoreResult.TotalScore {
				best[topic] = &vo.TopicRouteCandidate{
					TopicName: topic,
					Score:     scoreResult.TotalScore,
					Reason:    scoreResult.Reason,
					Source:    scoreResult.Source,
					Features:  scoreResult.Features,
				}
			}
		}
	}
	candidates := utils.MapValues(best)
	slices.SortFunc(candidates, func(a, b *vo.TopicRouteCandidate) int { return -cmp.Compare(a.Score, b.Score) })

	return utils.Limit(candidates, candidatesLimit), nil
}

// parseJsonStringArray 处理画像字段中的字符串数组：支持 ["a","b"]
func parseJsonStringArray(raw string) []string {
	cleaned := utils.Trim(raw)
	if cleaned == "" || cleaned == "[]" {
		return nil
	}

	// 优先尝试 JSON 解析
	var items []string
	if err := utils.Unmarshal(cleaned, &items); err == nil {
		return items
	}

	// 回退到手工解析
	inner := strings.TrimPrefix(strings.TrimSuffix(cleaned, "]"), "[")
	items = utils.Map(strings.Split(inner, ","), func(item string) string {
		return utils.Trim(strings.ReplaceAll(item, "\"", ""))
	})
	return utils.Filter(items, func(item string) bool {
		return utils.IsNotBlank(item)
	})
}
