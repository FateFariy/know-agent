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

const (
	documentCandidateLimit = 5
	lowConfidenceThreshold = 0.55
)

type DocumentRanker struct {
	repo       adapter.KnowledgeRepository
	docGateway adapter.DocumentGateway
	*base
}

func NewDocumentRanker(repo adapter.KnowledgeRepository, docGateway adapter.DocumentGateway, opts ...Option) *DocumentRanker {
	return &DocumentRanker{
		repo:       repo,
		docGateway: docGateway,
		base:       newBaseRanker(docGateway, opts...),
	}
}

// Rank 对文档进行打分，并将 top-score 的文档返回
//
// 打分策略：语义分 + 词面辅助分 + 关键词命中分 + 与 topScope 匹配加分 + 主题关系加分
// 低置信度扩展：当置信度 < 阈值时，将允许范围内的文档作为零分候选补充
func (r *DocumentRanker) Rank(ctx context.Context, rankCtx *Context) error {
	// 获取可检索文档列表，若无文档则直接返回
	documents, err := r.listRetrievableDocuments(ctx, rankCtx)
	if len(documents) == 0 {
		return nil
	}

	// 加载文档画像，用于构建路由文本和后续特征
	profiles, err := r.docGateway.FindDocumentProfiles(ctx)
	if err != nil {
		return fmt.Errorf("查询文档画像失败: %v", err)
	}
	profileByDoc := utils.MapBy(profiles, func(p *vo.DocumentProfile) (int64, *vo.DocumentProfile) {
		return p.DocumentId, p
	})

	// 加载主题-文档关系，用于主题关系加分（仅针对排名最高的主题）
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

	// 为每个文档构建路由文本（用于语义计算），过滤掉空文本的文档
	routeTexts := make([]string, 0, len(documents))
	k := 0
	for _, doc := range documents {
		text := doc.BuildRouteText(profileByDoc[doc.DocumentId])
		if text != "" {
			routeTexts = append(routeTexts, text)
			documents[k] = doc
			k++
		}
	}
	documents = documents[:k]
	// 计算语义分和词面辅助分
	semanticScores := r.computeSemanticScores(ctx, rankCtx, routeTexts)
	lexicalScores := r.searchLexicalScores(ctx, rankCtx, "document", documentCandidateLimit)

	// 逐一打分：基础分（语义+词面）+ 主题关系加分
	candidates := make(vo.DocumentRouteCandidates, 0, len(documents))
	for i, doc := range documents {
		// 基础分：语义分 + 词面分
		scoreResult := r.scorer.Score(&score.Features{
			SemanticScore: semanticScores[i],
			LexicalScore:  lexicalScores[doc.DocumentId],
		})

		// 主题关系加分：若存在最高主题且文档与该主题有关联得分，则累加（此处临时计算，后续正式重算）
		topicRelationScore := 0.0
		if topTopicId > 0 {
			if relMap, ok := relationByTopic[topTopicId]; ok {
				if rel, ok := relMap[doc.DocumentId]; ok && rel.RelationScore > 0 {
					topicRelationScore = rel.RelationScore
				}
			}
		}
		// 重新计算包含主题关系分的最终得分
		scoreResult = r.scorer.Score(&score.Features{
			SemanticScore: semanticScores[i],
			LexicalScore:  lexicalScores[doc.DocumentId],
			RelationScore: topicRelationScore,
		}, score.WithRelationWeight(20))

		// 过滤掉无效或非正分文档
		if scoreResult.TotalScore <= 0 {
			continue
		}
		candidates = append(candidates, &vo.DocumentRouteCandidate{
			DocumentId:      doc.DocumentId,
			DocumentName:    doc.DocumentName,
			LastIndexTaskId: doc.LastIndexTaskId,
			Score:           scoreResult.TotalScore,
			Reason:          scoreResult.Reason,
			Source:          scoreResult.Source,
			Features:        scoreResult.Features,
		})
	}

	// 按得分降序排列并截断至候选数限制
	slices.SortFunc(candidates, func(a, b *vo.DocumentRouteCandidate) int { return -cmp.Compare(a.Score, b.Score) })
	candidates = utils.Limit(candidates, documentCandidateLimit)
	rankCtx.DocumentCandidates = candidates

	if len(candidates) == 0 {
		rankCtx.Diagnostics["NO_EFFECTIVE_ROUTE_CANDIDATE"] = struct{}{}
		return nil
	}
	// 若置信度高于阈值，直接返回；否则执行低置信度扩展
	if candidates.ComputeConfidence() > lowConfidenceThreshold {
		return nil
	}

	// 低置信度扩展：在允许文档范围内补充零分候选，提高召回
	expanded := r.expandCandidatesWithAllowedScope(rankCtx, documents, documentCandidateLimit)
	if len(expanded) > len(candidates) {
		rankCtx.Diagnostics["LOW_CONFIDENCE_ALLOWED_SCOPE_EXPANSION"] = struct{}{}
	}
	rankCtx.DocumentCandidates = expanded

	return nil
}

// expandCandidatesWithAllowedScope 扩展排名候选，补充允许文档，生成有界候选池
// 当主排名候选置信度不足时，使用允许文档范围（AllowedDocumentIds）内的文档作为后备，
// 以零分加入，确保最终候选池非空且包含有界范围内的文档。
func (r *DocumentRanker) expandCandidatesWithAllowedScope(rankCtx *Context, documents []*vo.DocumentMetadata, limit int) []*vo.DocumentRouteCandidate {
	boundedLimit := max(limit, 1)
	merged := make([]*vo.DocumentRouteCandidate, 0, boundedLimit)
	rankedDocIds := make(map[int64]struct{})

	// 先加入已有的高排名候选（保留原顺序）
	for _, cand := range rankCtx.DocumentCandidates {
		if cand == nil || cand.DocumentId <= 0 {
			continue
		}
		if len(merged) >= boundedLimit {
			break
		}
		merged = append(merged, cand)
		rankedDocIds[cand.DocumentId] = struct{}{}
	}

	// 若已填满或无可用文档，直接返回
	if len(documents) == 0 || len(merged) >= boundedLimit {
		return merged
	}

	// 遍历所有可检索文档，补充未被选中的允许文档（以零分标记为后备）
	for _, doc := range documents {
		if doc == nil || doc.DocumentId == 0 || doc.LastIndexTaskId == 0 {
			continue
		}
		if _, exists := rankedDocIds[doc.DocumentId]; exists {
			continue
		}

		merged = append(merged, &vo.DocumentRouteCandidate{
			DocumentId:      doc.DocumentId,
			DocumentName:    doc.DocumentName,
			LastIndexTaskId: doc.LastIndexTaskId,
			Score:           0.0,
			Reason:          "未形成有效路由特征，按允许文档范围有界扩展候选池",
			Source:          "ALLOWED_DOCUMENT_SCOPE",
			Features: map[string]float64{
				"allowedScopeFallback": 1.0,
			},
		})
		rankedDocIds[doc.DocumentId] = struct{}{}
		if len(merged) >= boundedLimit {
			break
		}
	}

	return merged
}
