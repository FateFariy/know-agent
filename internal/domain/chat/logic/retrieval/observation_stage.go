package retrieval

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ObservationStage 观测数据记录阶段，记录通道执行与检索结果观测数据
type ObservationStage struct {
	repo adapter.ChatRepository
}

// NewObservationStage 创建观测数据记录阶段
func NewObservationStage(repo adapter.ChatRepository) *ObservationStage {
	return &ObservationStage{repo: repo}
}

func (s *ObservationStage) Name() string {
	return "Observation"
}

// Execute 记录通道执行观测与检索结果观测数据，结果写入 state.ObservationPersistence。
func (s *ObservationStage) Execute(ctx context.Context, state *RetrievalState) error {
	if len(state.ChannelResults) == 0 {
		return nil
	}

	trace := vo.TraceFromCtx(ctx)
	if trace == nil {
		return nil
	}

	if err := s.recordChannelObservations(ctx, trace, state); err != nil {
		logx.Warnf("记录通道观测数据失败: subQuestionIndex=%d, error=%v", state.Input.SubQuestionIndex, err)
	}

	if err := s.recordRetrievalResultObservations(ctx, trace, state); err != nil {
		logx.Warnf("记录检索结果观测数据失败: subQuestionIndex=%d, error=%v", state.Input.SubQuestionIndex, err)
	}
	return nil
}

// recordChannelObservations 记录渠道执行观测数据
func (s *ObservationStage) recordChannelObservations(ctx context.Context, trace *vo.ConversationTrace, state *RetrievalState) error {
	if len(state.ChannelResults) == 0 {
		return nil
	}

	end := time.Now()
	executions := make([]*entity.ChatChannelExecution, 0, len(state.ChannelResults))

	// 构建结果的映射
	resultsMap := make(map[string]*RetrievalChannelResult, len(state.ChannelResults))
	for _, r := range state.ChannelResults {
		resultsMap[r.Name] = r
	}

	// 构建渠道追踪的映射
	docMap := utils.GroupBy(state.FinalDocs, func(doc *vo.DocumentChunk) (string, *vo.DocumentChunk) {
		return doc.Channel, doc
	})

	for _, result := range state.ChannelResults {
		channelName := result.Name
		execution := &entity.ChatChannelExecution{
			ConversationId:     trace.ConversationId(),
			ExchangeId:         trace.ExchangeId(),
			TraceId:            trace.TraceId(),
			SubQuestionIndex:   state.Input.SubQuestionIndex,
			SubQuestion:        state.Input.SubQuestion,
			ChannelType:        channelName,
			StartTime:          state.Start,
			EndTime:            end,
			DurationMs:         end.Sub(state.Start).Milliseconds(),
			ExecutionState:     1,
			RecalledCount:      len(result.RawDocuments),
			AcceptedCount:      len(result.AcceptedDocuments),
			FinalSelectedCount: len(docMap[channelName]),
		}
		execution.SetScores(result.RawDocuments)
		executions = append(executions, execution)
	}

	return s.repo.InsertChannelExecutions(ctx, executions)
}

// recordRetrievalResultObservations 记录检索结果观测数据，返回观测持久化状态
func (s *ObservationStage) recordRetrievalResultObservations(ctx context.Context, trace *vo.ConversationTrace, state *RetrievalState) error {
	if len(state.ChannelResults) == 0 {
		return nil
	}

	results := s.projectRetrievalResults(trace, state)
	if err := s.repo.InsertRetrievalResults(ctx, results); err != nil {
		logx.Warnf("写入检索结果观测数据失败: subQuestionIndex=%d, error=%v",
			state.Input.SubQuestionIndex, err)
		return err
	}

	return nil
}

// projectRetrievalResults 将原始/过滤/融合/重排/最终文档投影为 ChatRetrievalResult 列表
func (s *ObservationStage) projectRetrievalResults(trace *vo.ConversationTrace, state *RetrievalState) []*entity.ChatRetrievalResult {
	// 构建最终文档 FinalRank 映射（按 ParentChunkId）
	finalRankMap := make(map[int64]int)
	for i, doc := range state.FinalDocs {
		finalRankMap[doc.ParentChunkId] = i + 1
	}

	// 按 RRFScore 降序排序 fusedDocs，构建 RrfRank 映射
	slices.SortFunc(state.FusedDocs, func(a, b *vo.DocumentChunk) int {
		return cmp.Compare(b.RRFScore, a.RRFScore)
	})
	rrfRankMap := make(map[string]int)
	rrfScoreMap := make(map[string]float64)
	for i, doc := range state.FusedDocs {
		rrfRankMap[doc.ID] = i + 1
		rrfScoreMap[doc.ID] = doc.RRFScore
	}

	rerankScoreMap := make(map[int64]float64)
	for _, doc := range state.RerankedDocs {
		rerankScoreMap[doc.ParentChunkId] = doc.RerankScore
	}

	// 构建"通过闸门"的文档 ID 集合（按渠道名分组）
	gatePassedSet := make(map[string]map[string]int)
	for _, fr := range state.ChannelResults {
		gatePassedSet[fr.Name] = make(map[string]int)
		for _, doc := range fr.AcceptedDocuments {
			gatePassedSet[fr.Name][doc.ID] = 1
		}
	}

	iteratee := func(acc int, r *RetrievalChannelResult) int { return acc + len(r.RawDocuments) }
	rawCandidateCount := utils.Reduce(state.ChannelResults, iteratee, 0)

	results := make([]*entity.ChatRetrievalResult, 0, rawCandidateCount)
	for _, result := range state.ChannelResults {
		channelName := result.Name
		for i, doc := range result.RawDocuments {
			view := &entity.ChatRetrievalResult{
				ConversationId:   trace.ConversationId(),
				ExchangeId:       trace.ExchangeId(),
				TraceId:          trace.TraceId(),
				SubQuestionIndex: state.Input.SubQuestionIndex,
				SubQuestion:      state.Input.SubQuestion,
				ChannelType:      channelName,
				ChannelRank:      i + 1,
				RrfRank:          rrfRankMap[doc.ID],
				RrfScore:         rrfScoreMap[doc.ID],
				RerankScore:      rerankScoreMap[doc.ParentChunkId],
				GatePassed:       gatePassedSet[channelName][doc.ID],
			}
			view.SetDocumentInfo(doc)

			// 判定是否被选入最终 Prompt
			if view.GatePassed == 0 {
				view.SelectionReason = s.resolveGateFilteredReason(state, channelName)
			} else if rank, ok := finalRankMap[doc.ParentChunkId]; ok {
				view.IsSelected = 1
				view.FinalRank = rank
				view.SelectionReason = "已选入最终 Prompt"
			} else {
				view.SelectionReason = fmt.Sprintf("超出 finalTopK 限制（topK=%d）", state.Plan.FinalTopK)
			}

			results = append(results, view)
		}
	}

	return results
}

// resolveGateFilteredReason 根据渠道类型返回闸门过滤原因
func (s *ObservationStage) resolveGateFilteredReason(state *RetrievalState, channelName string) string {
	channel, _ := state.Input.RequireChannel(channelName)
	switch channelName {
	case enum.RetrievalChannelVector:
		return fmt.Sprintf("向量闸门过滤：分数 < 阈值 %.4f", channel.MinimumScore)
	case enum.RetrievalChannelKeyword:
		return fmt.Sprintf("关键词闸门过滤：分数低于相对阈值（floor=%.2f）", channel.RelativeScoreFloor)
	default:
		return "闸门过滤"
	}
}
