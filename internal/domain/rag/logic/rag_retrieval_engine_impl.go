package logic

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	kl "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const rrfK = 60

type RagRetrievalEngine struct {
	channels                  []RetrievalChannel
	documentKnowledgeLogic    kl.DocumentKnowledgeLogic
	channelTimeout            time.Duration
	subQuestionTimeout        time.Duration
	minVectorSimilarity       float64
	keywordRelativeScoreFloor float64
	candidateTopK             int
	parentEvidenceMaxChars    int
	rerankEnabled             bool
	finalTopK                 int
}

func NewRagRetrievalEngine(svcCtx *svc.ServiceContext, channels []RetrievalChannel, documentKnowledgeLogic kl.DocumentKnowledgeLogic) *RagRetrievalEngine {
	return &RagRetrievalEngine{
		channels:                  channels,
		documentKnowledgeLogic:    documentKnowledgeLogic,
		subQuestionTimeout:        svcCtx.Config.Chat.Rag.SubQuestionTimeout,
		channelTimeout:            svcCtx.Config.Chat.Rag.ChannelTimeout,
		minVectorSimilarity:       svcCtx.Config.Chat.Rag.MinVectorSimilarity,
		keywordRelativeScoreFloor: svcCtx.Config.Chat.Rag.KeywordRelativeScoreFloor,
		candidateTopK:             svcCtx.Config.Chat.Rag.CandidateTopK,
		parentEvidenceMaxChars:    svcCtx.Config.Chat.Rag.ParentEvidenceMaxChars,
		rerankEnabled:             svcCtx.Config.Chat.Rag.RerankEnabled,
		finalTopK:                 svcCtx.Config.Chat.Rag.FinalTopK,
	}
}

var _ RagRetriever = (*RagRetrievalEngine)(nil)

func (e *RagRetrievalEngine) Retrieve(ctx context.Context, plan *cvo.ConversationExecutionPlan, tracer *cvo.ConversationTrace) (*vo.RagRetrievalContext, error) {
	ragCtx := vo.NewRagRetrievalContext(plan.RetrievalQuestion)

	subQuestions := plan.RetrievalSubQuestions
	if len(subQuestions) == 0 {
		subQuestions = []string{plan.RetrievalQuestion}
	}

	evidenceList := e.retrieveSubQuestionParallel(ctx, ragCtx, subQuestions, plan, tracer)
	acceptedCount := stream.FromSlice(evidenceList).
		Filter(func(item *vo.SubQuestionEvidence) bool { return item != nil && len(item.Documents) > 0 }).
		Count()

	logx.Infof("RAG 检索完成: retrievalQuestion='%s', originalSubQuestionCount=%d, acceptedSubQuestionCount=%d, notes=%v",
		plan.RetrievalQuestion, len(evidenceList), acceptedCount, ragCtx.RetrievalNotes)

	e.assignReferenceIds(evidenceList)
	ragCtx.SubQuestionEvidenceList = evidenceList

	return ragCtx, nil
}

func (e *RagRetrievalEngine) retrieveSubQuestionParallel(ctx context.Context, ragCtx *vo.RagRetrievalContext, subQuestions []string,
	plan *cvo.ConversationExecutionPlan, tracer *cvo.ConversationTrace) []*vo.SubQuestionEvidence {
	timeoutCtx, cancel := context.WithTimeout(ctx, e.subQuestionTimeout)
	defer cancel()

	resultChan := make(chan *vo.SubQuestionEvidence, len(subQuestions))
	defer close(resultChan)

	// todo 线程池改造
	for i, sq := range subQuestions {
		go func(subQuestionIndex int, subQuestion string) {
			channelResults := e.retrieveChannelParallel(timeoutCtx, ragCtx, subQuestionIndex, subQuestion, plan)

			if len(channelResults) == 0 {
				ragCtx.AddRetrievalNotef("子问题%d没有可用的检索通道。", subQuestionIndex)
				resultChan <- &vo.SubQuestionEvidence{SubQuestionIndex: subQuestionIndex, SubQuestion: subQuestion}
				return
			}

			rawChannelResults := slice.Filter(channelResults, func(index int, result *RetrievalChannelResult) bool {
				return len(result.Documents) > 0
			})
			filteredResults := slice.Map(rawChannelResults, func(index int, result *RetrievalChannelResult) *RetrievalChannelResult {
				return e.applyEvidenceGate(result)
			})

			channelTraces := e.buildChannelTraces(rawChannelResults, filteredResults)

			for _, r := range filteredResults {
				if len(r.Documents) > 0 {
					ragCtx.AddUsedChannel(r.ChannelName)
				}
			}

			documents := e.fuseByRRF(filteredResults)
			parentSearchDocs, err := e.documentKnowledgeLogic.ElevateToParentBlocks(timeoutCtx, documents, e.parentEvidenceMaxChars)
			if err != nil {
				Warnf("父块提升失败: subQuestionIndex=%d, error=%v", subQuestionIndex, err)
				return
			}

			rerankedCandidates := e.applyRerank(ragCtx, parentSearchDocs, subQuestion)

			finalTopK := min(e.finalTopK, len(rerankedCandidates))
			finalDocuments := rerankedCandidates[:finalTopK]

			ragCtx.AddRetrievalNotef(fmt.Sprintf("子问题%d检索完成：%s，final=%d",
				subQuestionIndex, e.summarizeChannelResults(filteredResults), len(finalDocuments)))

			fusedCount := len(documents)
			parentCount := len(parentSearchDocs)
			rerankedCount := len(rerankedCandidates)

			// 记录观测数据
			if tracer != nil {
				e.recordChannelObservations(tracer, subQuestionIndex, subQuestion, rawChannelResults, filteredResults, channelTraces)
				e.recordRetrievalResultObservations(tracer, subQuestionIndex, subQuestion, rawChannelResults, filteredResults, documents, rerankedCandidates, finalDocuments)
			}

			resultChan <- &vo.SubQuestionEvidence{
				SubQuestionIndex:       subQuestionIndex,
				SubQuestion:            subQuestion,
				Documents:              finalDocuments,
				References:             nil,
				ChannelTraces:          channelTraces,
				FusedCandidateCount:    fusedCount,
				ParentCandidateCount:   parentCount,
				RerankedCandidateCount: rerankedCount,
			}
		}(i+1, sq)
	}

	// 收集所有子问题的结果
	evidenceList := make([]*vo.SubQuestionEvidence, 0, len(subQuestions))
	for i := 0; i < len(subQuestions); i++ {
		evidenceList = append(evidenceList, <-resultChan)
	}
	return evidenceList

}

// retrieveChannelParallel 并行检索单个子问题的所有通道
// 使用goroutine并发执行各通道检索，通过超时控制防止阻塞，失败自动降级返回空结果
func (e *RagRetrievalEngine) retrieveChannelParallel(ctx context.Context, ragCtx *vo.RagRetrievalContext, subQuestionIndex int, subQuestion string, plan *cvo.ConversationExecutionPlan) []*RetrievalChannelResult {
	// 创建带超时的上下文，超时时间为通道超时配置
	timeoutCtx, cancel := context.WithTimeout(ctx, e.channelTimeout)
	defer cancel()

	// 过滤仅执行支持当前计划的通道
	channels := slice.Filter(e.channels, func(_ int, item RetrievalChannel) bool { return item.Supports(plan) })

	// 创建结果通道，缓冲大小为通道数量，避免goroutine阻塞
	resultCh := make(chan *RetrievalChannelResult, len(channels))
	defer close(resultCh)

	// 遍历所有通道，并行启动检索
	for _, channel := range channels {
		go func(ch RetrievalChannel) {
			result, err := ch.Retrieve(timeoutCtx, subQuestion, plan)
			if err != nil {
				Warnf("检索通道失败: subQuestionIndex=%d, subQuestion='%s', channel='%s', error=%v",
					subQuestionIndex, subQuestion, ch.ChannelName(), err)
				ragCtx.AddRetrievalNotef("子问题%d通道[%s]检索失败或超时，已自动降级。", subQuestionIndex, ch.ChannelName())
				result = &RetrievalChannelResult{ChannelName: ch.ChannelName(), Documents: nil}
			}
			resultCh <- result
		}(channel)
	}

	// 收集结果（或超时退出），确保不会无限等待
	channelResults := make([]*RetrievalChannelResult, 0, len(channels))
	for {
		select {
		case result := <-resultCh:
			channelResults = append(channelResults, result)
			if len(channelResults) == len(channels) {
				return channelResults
			}
		case <-timeoutCtx.Done():
			return channelResults
		}
	}
}

// applyEvidenceGate 根据通道类型应用不同的分数过滤策略，过滤掉置信度不足的文档
func (e *RagRetrievalEngine) applyEvidenceGate(result *RetrievalChannelResult) *RetrievalChannelResult {
	if result == nil || len(result.Documents) == 0 {
		return result
	}

	var documents []*klvo.Document
	switch result.ChannelName {
	case vo.RetrievalChannelVector:
		// 向量通道：使用绝对相似度阈值过滤
		documents = slice.Filter(result.Documents, func(index int, doc *klvo.Document) bool {
			return doc.Score >= e.minVectorSimilarity
		})
	case vo.RetrievalChannelKeyword:
		// 关键词通道：使用相对分数阈值过滤（相对于最高分）
		maxScore := slices.MaxFunc(result.Documents, func(doc1, doc2 *klvo.Document) int { return int(doc1.Score - doc2.Score) }).Score
		documents = slice.Filter(result.Documents, func(index int, doc *klvo.Document) bool {
			return doc.Score >= (e.keywordRelativeScoreFloor * maxScore)
		})
	default:
		documents = result.Documents
	}

	return &RetrievalChannelResult{
		ChannelName: result.ChannelName,
		Documents:   documents,
	}
}

type candidateHolder struct {
	document *klvo.Document
	score    float64
	channels map[string]struct{}
}

// fuseByRRF 融合多个通道的候选结果（基于RRF算法）
// RRF(Reciprocal Rank Fusion)通过合并各通道的排名信息，计算综合分数，实现多通道结果融合
func (e *RagRetrievalEngine) fuseByRRF(channelResults []*RetrievalChannelResult) []*klvo.Document {
	var holders []*candidateHolder

	// 遍历所有通道结果，累积计算RRF分数
	for _, channelResult := range channelResults {
		holders = e.accumulateRRF(channelResult)
	}

	// 按RRF分数降序排序，取前N个文档
	result := make([]*klvo.Document, 0, len(holders))
	stream.FromSlice(holders).
		Sorted(func(a, b *candidateHolder) bool { return a.score > b.score }).
		Limit(e.candidateTopK).
		ForEach(func(holder *candidateHolder) {
			// 填充文档元数据（分数、通道来源）
			if holder.document.Meta == nil {
				holder.document.Meta = make(map[string]any)
			}
			holder.document.Score = holder.score
			holder.document.Meta[klvo.MetaRRFScore] = holder.score
			holder.document.Meta[klvo.MetaChannel] = utils.Ternary(len(holder.channels) > 1, vo.RetrievalChannelHybrid, maputil.Keys(holder.channels)[0])
			result = append(result, holder.document)
		})
	return result
}

// accumulateRRF 计算文档的RRF分数
func (e *RagRetrievalEngine) accumulateRRF(channelResult *RetrievalChannelResult) []*candidateHolder {
	holders := make(map[string]*candidateHolder)
	for rank, doc := range channelResult.Documents {
		rrfScore := 1.0 / float64(rrfK+rank+1)
		holder, ok := holders[doc.ID]
		if !ok {
			holder = &candidateHolder{
				document: doc,
				channels: make(map[string]struct{}),
			}
			holders[doc.ID] = holder
		}
		holder.score += rrfScore
		holder.channels[channelResult.ChannelName] = struct{}{}
	}
	return maputil.Values(holders)
}

func (e *RagRetrievalEngine) applyRerank(ragCtx *vo.RagRetrievalContext, candidates []*klvo.Document, subQuestion string) []*klvo.Document {
	if !e.rerankEnabled || len(candidates) == 0 || e.rerankPostProcessor == nil {
		return candidates
	}

	ragCtx.AddUsedChannel(vo.RetrievalChannelRerank)
	result, err := e.rerankPostProcessor.Process(subQuestion, candidates)
	if err != nil {
		Warnf("重排序处理失败: subQuestion='%s', error=%v", subQuestion, err)
		return candidates
	}
	return result
}

func (e *RagRetrievalEngine) assignReferenceIds(evidenceList []*vo.SubQuestionEvidence) {
	referenceNumber := 1
	assignedIDs := make(map[string]string)

	for _, evidence := range evidenceList {
		references := make([]*vo.SearchReference, 0, len(evidence.Documents))
		for _, doc := range evidence.Documents {
			ref := SearchReferenceMapper(doc, evidence.SubQuestionIndex, evidence.SubQuestion, 0)
			uniqueKey := UniqueKey(ref)

			assignedID, ok := assignedIDs[uniqueKey]
			if !ok {
				assignedID = fmt.Sprintf("%d", referenceNumber)
				assignedIDs[uniqueKey] = assignedID
				referenceNumber++
			}
			ref.ReferenceId = assignedID
			references = append(references, ref)
		}
		evidence.References = references
	}
}

func (e *RagRetrievalEngine) summarizeChannelResults(channelResults []*RetrievalChannelResult) string {
	if len(channelResults) == 0 {
		return "没有启用任何检索通道"
	}
	parts := slice.Map(channelResults, func(_ int, result *RetrievalChannelResult) string {
		return fmt.Sprintf("%s=%d", result.ChannelName, len(result.Documents))
	})
	return strings.Join(parts, "，")
}

// buildChannelTraces 构建子问题渠道执行追踪
// 统计每个检索渠道的召回数量（原始结果）和接受数量（过滤后结果），用于追踪检索效果
func (e *RagRetrievalEngine) buildChannelTraces(rawResults, filteredResults []*RetrievalChannelResult) []*vo.SubQuestionChannelTrace {
	if len(rawResults) == 0 && len(filteredResults) == 0 {
		return nil
	}

	// 初始化统计映射，key为渠道名称，value为文档数量
	rawMap := make(map[string]int)
	filteredMap := make(map[string]int)
	channelNames := make(map[string]struct{})

	// 统计原始检索结果中各渠道的文档数量
	slice.ForEach(rawResults, func(index int, r *RetrievalChannelResult) {
		rawMap[r.ChannelName] = len(r.Documents)
		channelNames[r.ChannelName] = struct{}{}
	})

	// 统计过滤后结果中各渠道的文档数量
	slice.ForEach(filteredResults, func(index int, r *RetrievalChannelResult) {
		filteredMap[r.ChannelName] = len(r.Documents)
		channelNames[r.ChannelName] = struct{}{}
	})

	// 遍历所有渠道，构建追踪记录（缺失的数量默认为0）
	return slice.Map(maputil.Keys(channelNames), func(index int, name string) *vo.SubQuestionChannelTrace {
		return &vo.SubQuestionChannelTrace{
			ChannelName:   name,
			RecalledCount: rawMap[name],
			AcceptedCount: filteredMap[name],
		}
	})
}

func Warnf(format string, args ...interface{}) {
	logx.Alert(fmt.Sprintf(format, args...))
}

// recordChannelObservations 记录渠道执行观测数据，包括召回数量、接受数量、分数等
func (e *RagRetrievalEngine) recordChannelObservations(tracer *cvo.ConversationTrace, subQuestionIndex int, subQuestion string,
	rawResults, filteredResults []*RetrievalChannelResult, channelTraces []*vo.SubQuestionChannelTrace) error {
	if len(rawResults) == 0 {
		return nil
	}

	executions := make([]*vo.ChannelExecutionView, 0, len(rawResults))
	filteredResultsMap := utils.SliceToMapBy(filteredResults, func(r *RetrievalChannelResult) (string, *RetrievalChannelResult) {
		return r.ChannelName, r
	})
	channelTracesMap := utils.SliceToMapBy(channelTraces, func(t *vo.SubQuestionChannelTrace) (string, *vo.SubQuestionChannelTrace) {
		return t.ChannelName, t
	})

	for _, rawResult := range rawResults {
		channelName := rawResult.ChannelName
		execution := &vo.ChannelExecutionView{
			ExchangeId:       tracer.GetExchangeId(),
			TraceId:          tracer.GetTraceId(),
			SubQuestionIndex: subQuestionIndex,
			SubQuestion:      subQuestion,
			ChannelType:      channelName,
			ExecutionState:   1,
			RecalledCount:    len(rawResult.Documents),
		}
		// 获取过滤后的结果
		if filteredResult, ok := filteredResultsMap[channelName]; ok {
			execution.AcceptedCount = len(filteredResult.Documents)
		}
		// 获取追踪记录
		if trace, ok := channelTracesMap[channelName]; ok {
			execution.FinalSelectedCount = trace.AcceptedCount
		}
		execution.SetScores(rawResult.Documents)
		executions = append(executions, execution)
	}

	// todo 待完善最终保存
	tracer.RecordChannelExecutions(executions)
	return nil
}

// recordRetrievalResultObservations 记录检索结果观测数据，包括各阶段分数、是否通过闸门、是否被选中等
func (e *RagRetrievalEngine) recordRetrievalResultObservations(tracer *cvo.ConversationTrace, subQuestionIndex int, subQuestion string,
	rawResults, filteredResults []*RetrievalChannelResult, mergedCandidates, rerankedCandidates, finalDocuments []*klvo.Document) error {
	if len(rawResults) == 0 {
		return nil
	}

	// 构建最终文档ID到排名的映射
	finalRankMap := make(map[string]int)
	for i, doc := range finalDocuments {
		if doc.ID != "" {
			finalRankMap[doc.ID] = i + 1
		}
	}

	// 构建通过闸门的文档ID集合
	gatePassedSet := make(map[string]bool)
	for _, fr := range filteredResults {
		if fr.Documents != nil {
			for _, doc := range fr.Documents {
				if doc.ID != "" {
					gatePassedSet[doc.ID] = true
				}
			}
		}
	}

	results := make([]*vo.RetrievalResultView, 0)

	for _, rawResult := range rawResults {
		channelName := rawResult.ChannelName
		for i, doc := range rawResult.Documents {
			view := &vo.RetrievalResultView{
				ExchangeId:       tracer.GetExchangeId(),
				TraceId:          tracer.GetTraceId(),
				SubQuestionIndex: subQuestionIndex,
				SubQuestion:      subQuestion,
				ChannelType:      channelName,
				ChannelRank:      i + 1,
			}

			view.SetDocumentInfo(doc)

			// 设置原始分数
			view.OriginalScore = doc.Score

			// 从Meta中提取RRF分数和重排序分数
			if doc.Meta != nil {
				if val, ok := doc.Meta[klvo.MetaRRFScore]; ok {
					if score, ok := val.(float64); ok {
						view.RrfScore = score
					}
				}
				if val, ok := doc.Meta["rerankScore"]; ok {
					if score, ok := val.(float64); ok {
						view.RerankScore = score
					}
				}
			}

			// 判断是否通过闸门
			if doc.ID != "" {
				view.GatePassed = gatePassedSet[doc.ID]
			}

			// 判断是否被选中
			if doc.ID != "" {
				if rank, ok := finalRankMap[doc.ID]; ok {
					view.Selected = true
					view.FinalRank = rank
					view.SelectionReason = "已选入最终 Prompt"
				} else if !view.GatePassed {
					// 未通过闸门
					if "vector" == channelName {
						view.SelectionReason = fmt.Sprintf("向量闸门过滤：分数 %.4f < 阈值 %.4f",
							view.OriginalScore, e.minVectorSimilarity)
					} else if "keyword" == channelName {
						view.SelectionReason = fmt.Sprintf("关键词闸门过滤：分数 %.4f 低于相对阈值（floor=%.2f）",
							view.OriginalScore, e.keywordRelativeScoreFloor)
					} else {
						view.SelectionReason = fmt.Sprintf("闸门过滤：分数 %.4f", view.OriginalScore)
					}
				} else {
					// 超出finalTopK限制
					view.SelectionReason = fmt.Sprintf("超出 finalTopK 限制（topK=%d）", e.finalTopK)
				}
			}

			results = append(results, view)
		}
	}

	tracer.RecordRetrievalResults(results)
	return nil
}
