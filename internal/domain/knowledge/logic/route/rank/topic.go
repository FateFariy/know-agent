package rank

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/swiftbit/know-agent/common/logx"
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
	repo       adapter.KnowledgeRepository
	docGateway adapter.DocumentGateway
	scorer     score.Scorer
	base
}

func NewTopicRanker(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway, embedder adapter.Embedder,
	lexicalIndex adapter.RouteLexicalIndex, scorer score.Scorer) *TopicRanker {
	return &TopicRanker{
		repo:       repo,
		docGateway: docGateway,
		scorer:     scorer,
		base:       newBaseRanker(embedder, lexicalIndex),
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
	lexicalScores, _ := r.searchLexicalScores(ctx, rankCtx.RoutingText, "topic", 8)

	candidates := make([]*vo.TopicRouteCandidate, 0, len(nodes))
	for i, node := range nodes {
		scoreResult := r.scorer.Score(&score.Features{
			SemanticScore: semanticScores[i],
			LexicalScore:  lexicalScores[node.ID],
			RelationScore: utils.Ternary(preferredScopes[node.ScopeId], 1.0, 0.0),
		})
		if scoreResult.TotalScore > 0 {
			candidates = append(candidates, &vo.TopicRouteCandidate{
				TopicName: node.TopicName,
				Score:     scoreResult.TotalScore,
				Reason:    scoreResult.Reason,
				Source:    scoreResult.Source,
				Features:  scoreResult.Features,
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	rankCtx.TopicCandidates = candidates
	return nil
}

// deriveTopicsFromProfiles 当 topic 节点未配置时，按文档画像的 CoreTopics 派生主题候选
func (r *TopicRanker) deriveTopicsFromProfiles(ctx context.Context, rankCtx *Context) ([]*vo.TopicRouteCandidate, error) {
	docs, _ := r.listRetrievableDocuments(ctx, rankCtx)
	if len(docs) == 0 {
		return nil, nil
	}

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
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })

	return utils.Limit(candidates, candidatesLimit), nil
}

// listRetrievableDocuments 查询可检索的文档
func (r *TopicRanker) listRetrievableDocuments(ctx context.Context, rankCtx *Context) ([]*vo.DocumentMetadata, error) {
	var docs []*vo.DocumentMetadata
	var err error
	if len(rankCtx.AllowedDocumentIds) != 0 {
		docs, err = r.docGateway.FindRetrieveDocumentByIds(ctx, rankCtx.AllowedDocumentIds)
		if err != nil {
			logx.Warnf("查询可检索文档失败: %v", err)
		}
	}
	if len(docs) == 0 {
		docs, err = r.docGateway.FindRetrievableByKbIds(ctx, rankCtx.SelectedKnowledgeBaseIds)
		if err != nil {
			logx.Warnf("查询可检索文档失败: %v", err)
		}
	}
	return docs, err
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
