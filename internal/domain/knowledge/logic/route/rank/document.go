package rank

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/adapter"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route/score"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
)

const (
	documentCandidateLimit = 5
)

type DocumentRanker struct {
	repo       adapter.KnowledgeRepository
	docGateway adapter.DocumentGateway
	scorer     score.Scorer
	base
}

func NewDocumentRanker(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway, embedder adapter.Embedder,
	lexicalIndex adapter.RouteLexicalIndex, scorer score.Scorer) *DocumentRanker {
	return &DocumentRanker{
		repo:       repo,
		docGateway: docGateway,
		scorer:     scorer,
		base:       newBaseRanker(docGateway, embedder, lexicalIndex),
	}
}

// Rank 对文档进行打分，并将 top-score 的文档返回
//
// 打分策略：语义分 + 词面辅助分 + 关键词命中分 + 与 topScope 匹配加分 + 主题关系加分
// 低置信度扩展：当置信度 < 阈值时，将允许范围内的文档作为零分候选补充
func (r *DocumentRanker) Rank(ctx context.Context, rankCtx *Context) error {
	documents, err := r.listRetrievableDocuments(ctx, rankCtx)
	if len(documents) == 0 {
		return nil
	}

	// 加载文档画像
	profiles, err := r.docGateway.FindDocumentProfiles(ctx)
	if err != nil {
		return fmt.Errorf("查询文档画像失败: %v", err)
	}
	profileByDoc := utils.MapBy(profiles, func(p *vo.DocumentProfile) (int64, *vo.DocumentProfile) {
		return p.DocumentId, p
	})

	// 加载主题-文档关系
	where := make(map[string]any, 2)
	if len(rankCtx.SelectedKnowledgeBaseIds) > 0 {
		where["knowledge_base_id"] = rankCtx.SelectedKnowledgeBaseIds
	}
	if len(rankCtx.AllowedDocumentIds) > 0 {
		where["document_id"] = rankCtx.AllowedDocumentIds
	}
	relations, err := r.repo.SelectTopicDocumentRelations(ctx, where)
	if err != nil {
		return fmt.Errorf("查询 topic-document 关系失败: %v", err)
	}
	relationByTopic := make(map[int64]map[int64]*entity.KnowledgeTopicDocumentRelation, len(relations))
	for _, rel := range relations {
		if _, ok := relationByTopic[rel.TopicId]; !ok {
			relationByTopic[rel.TopicId] = make(map[int64]*entity.KnowledgeTopicDocumentRelation)
		}
		relationByTopic[rel.TopicId][rel.DocumentId] = rel
	}

	topTopicId := int64(0)
	if len(rankCtx.TopicCandidates) > 0 {
		topTopicId = rankCtx.TopicCandidates[0].TopicId
	}

	// 为每个文档准备文本与语义分
	routeTexts := make([]string, 0, len(documents))
	for _, doc := range documents {
		routeTexts = append(routeTexts, doc.BuildRouteText(profileByDoc[doc.DocumentId]))
	}
	semanticScores := r.computeSemanticScores(ctx, rankCtx, routeTexts)
	lexicalScores := r.searchLexicalScores(ctx, rankCtx, "document", documentCandidateLimit)

	// 打分：语义分 + 词面辅助分 + 关键词命中分
	matched := make([]*vo.DocumentRouteCandidate, 0, len(documents))
	for i, doc := range documents {
		routeText := routeTexts[i]
		semantic := 0.0
		if i < len(semanticScores) {
			semantic = semanticScores[i]
		}
		scoreResult := r.scorer.Score(&score.Features{
			SemanticScore: semantic,
			LexicalScore:  lexicalScores[doc.DocumentId],
		})

		tempScore := scoreResult.TotalScore
		// 主题关系加分
		topicRelationScore := 0.0
		if topTopicId > 0 {
			if relMap, ok := relationByTopic[topTopicId]; ok {
				if rel, ok := relMap[doc.DocumentId]; ok && rel.RelationScore > 0 {
					topicRelationScore = rel.RelationScore
					tempScore += topicRelationScore * 20
				}
			}
		}
		scoreResult = r.scorer.Score(&score.Features{
			SemanticScore: semantic,
			LexicalScore:  lexicalScores[doc.DocumentId],
			RelationScore: topicRelationScore,
		})

		if scoreResult.TotalScore <= 0 {
			continue
		}
		matched = append(matched, &vo.DocumentRouteCandidate{
			DocumentId:      doc.DocumentId,
			DocumentName:    doc.DocumentName,
			LastIndexTaskId: doc.LastIndexTaskId,
			Score:           scoreResult.TotalScore,
			Reason:          scoreResult.Reason,
			Source:          scoreResult.Source,
			Features:        scoreResult.Features,
		})
	}

	// 按得分降序排列
	slices.SortFunc(matched, func(a, b *vo.DocumentRouteCandidate) int { return -cmp.Compare(a.Source, b.Source) })
	candidates := utils.Limit(matched, documentCandidateLimit)

	if len(candidates) == 0 {
		rankCtx.Diagnostics["NO_EFFECTIVE_ROUTE_CANDIDATE"] = struct{}{}
		return nil
	}

	// 低置信度扩展：当无候选或置信度低时，补充允许范围内的文档
	if len(candidates) == 0 || (len(candidates) > 0 && r.computeConfidence(candidates) < lowConfidenceThreshold) {
		expanded := r.expandCandidatesWithAllowedScope(candidates, documents, q, documentCandidateLimit)
		if len(expanded) > len(candidates) {
			q.Diagnostics["LOW_CONFIDENCE_ALLOWED_SCOPE_EXPANSION"] = struct{}{}
		}
		candidates = expanded
	}

	return
}
