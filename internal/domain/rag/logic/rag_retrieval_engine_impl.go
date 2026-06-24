package logic

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	kl "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	klvo "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"
	rvo "github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
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
	}
}

var _ RagRetriever = (*RagRetrievalEngine)(nil)

func (e *RagRetrievalEngine) Retrieve(ctx context.Context, plan *vo.ConversationExecutionPlan, tracer *vo.ConversationTrace) (*rvo.RagRetrievalContext, error) {
	ragCtx := rvo.NewRagRetrievalContext(plan.RetrievalQuestion)

	subQuestions := plan.RetrievalSubQuestions
	if len(subQuestions) == 0 {
		subQuestions = []string{plan.RetrievalQuestion}
	}

	evidenceList := e.retrieveSubQuestionParallel(ctx, ragCtx, subQuestions, plan, tracer)
	acceptedCount := stream.FromSlice(evidenceList).
		Filter(func(item *rvo.SubQuestionEvidence) bool { return item != nil && len(item.Documents) > 0 }).
		Count()

	logx.Infof("RAG 检索完成: retrievalQuestion='%s', originalSubQuestionCount=%d, acceptedSubQuestionCount=%d, notes=%v",
		plan.RetrievalQuestion, len(evidenceList), acceptedCount, ragCtx.RetrievalNotes)

	e.assignReferenceIds(evidenceList)
	ragCtx.SubQuestionEvidenceList = evidenceList

	return ragCtx, nil
}

func (e *RagRetrievalEngine) retrieveSubQuestionParallel(ctx context.Context, ragCtx *rvo.RagRetrievalContext, subQuestions []string,
	plan *vo.ConversationExecutionPlan, tracer *vo.ConversationTrace) []*rvo.SubQuestionEvidence {
	timeoutCtx, cancel := context.WithTimeout(ctx, e.subQuestionTimeout)
	defer cancel()

	// todo 线程池改造
	resultChan := make(chan *rvo.SubQuestionEvidence, len(subQuestions))
	for i, sq := range subQuestions {
		go func(subQuestionIndex int, subQuestion string) {
			channelResults := e.retrieveChannelParallel(timeoutCtx, ragCtx, subQuestionIndex, subQuestion, plan)

			if len(channelResults) == 0 {
				ragCtx.AddRetrievalNotef("子问题%d没有可用的检索通道。", subQuestionIndex)
				resultChan <- &rvo.SubQuestionEvidence{SubQuestionIndex: subQuestionIndex, SubQuestion: subQuestion}
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
			}

			rerankedCandidates := e.applyRerank(subQuestion, parentCandidates, usedChannels)

			finalTopK := e.properties.FinalTopK
			if finalTopK <= 0 {
				finalTopK = 5
			}
			finalDocuments := make([]*ragCtx.DocumentCandidate, 0, len(rerankedCandidates))
			for i, doc := range rerankedCandidates {
				if i >= finalTopK {
					break
				}
				finalDocuments = append(finalDocuments, doc)
			}

			ragCtx.AddRetrievalNotef(fmt.Sprintf("子问题%d检索完成：%s，final=%d",
				subQuestionIndex, e.summarizeChannelResults(filteredResults), len(finalDocuments)))

			fusedCount := len(mergedCandidates)
			parentCount := len(parentCandidates)
			rerankedCount := len(rerankedCandidates)

			// return &ragCtx.SubQuestionEvidence{
			// 	SubQuestionIndex:       subQuestionIndex,
			// 	SubQuestion:            subQuestion,
			// 	Documents:              finalDocuments,
			// 	References:             nil,
			// 	ChannelTraces:          channelTraces,
			// 	FusedCandidateCount:    &fusedCount,
			// 	ParentCandidateCount:   &parentCount,
			// 	RerankedCandidateCount: &rerankedCount,
			// }
		}(i+1, sq)
	}

}

// retrieveChannelParallel 并行检索单个子问题的所有通道
// 使用goroutine并发执行各通道检索，通过超时控制防止阻塞，失败自动降级返回空结果
func (e *RagRetrievalEngine) retrieveChannelParallel(ctx context.Context, ragCtx *rvo.RagRetrievalContext, subQuestionIndex int, subQuestion string, plan *vo.ConversationExecutionPlan) []*RetrievalChannelResult {
	// 创建带超时的上下文，超时时间为通道超时配置
	timeoutCtx, cancel := context.WithTimeout(ctx, e.channelTimeout)
	defer cancel()

	// 创建结果通道，缓冲大小为通道数量，避免goroutine阻塞
	resultCh := make(chan *RetrievalChannelResult, len(e.channels))
	defer close(resultCh)

	// 遍历所有通道，并行启动检索（仅执行支持当前计划的通道）
	for _, channel := range e.channels {
		if channel.Supports(plan) {
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
	}

	// 收集结果（或超时退出），确保不会无限等待
	channelResults := make([]*RetrievalChannelResult, 0, len(e.channels))
	for {
		select {
		case result := <-resultCh:
			channelResults = append(channelResults, result)
		case <-timeoutCtx.Done():
			return channelResults
		}
	}
}

// applyEvidenceGate 应用证据门限过滤
// 根据通道类型应用不同的分数过滤策略，过滤掉置信度不足的文档
func (e *RagRetrievalEngine) applyEvidenceGate(result *RetrievalChannelResult) *RetrievalChannelResult {
	if result == nil || len(result.Documents) == 0 {
		return result
	}

	var documents []*klvo.Document
	switch result.ChannelName {
	case "vector":
		// 向量通道：使用绝对相似度阈值过滤
		documents = slice.Filter(result.Documents, func(index int, doc *klvo.Document) bool {
			return doc.Score >= e.minVectorSimilarity
		})
	case "keyword":
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
	holders := make(map[string]*candidateHolder)

	// 遍历所有通道结果，累积计算RRF分数
	for _, channelResult := range channelResults {
		e.accumulateRRF(channelResult, holders)
	}

	// 按RRF分数降序排序，取前N个文档
	result := make([]*klvo.Document, 0, len(holders))
	stream.FromSlice(maputil.Values(holders)).
		Sorted(func(a, b *candidateHolder) bool { return a.score > b.score }).
		Limit(e.candidateTopK).
		ForEach(func(holder *candidateHolder) {
			// 填充文档元数据（分数、通道来源）
			if holder.document.Meta == nil {
				holder.document.Meta = make(map[string]any)
			}
			holder.document.Meta[klvo.MetaScore] = holder.score
			holder.document.Meta[klvo.MetaRRFScore] = holder.score
			holder.document.Meta[klvo.MetaChannel] = utils.Ternary(len(holder.channels) > 1, "hybrid", maputil.Keys(holder.channels)[0])
			result = append(result, holder.document)
		})
	return result
}

// accumulateRRF 计算文档的RRF分数
func (e *RagRetrievalEngine) accumulateRRF(channelResult *RetrievalChannelResult, holders map[string]*candidateHolder) {
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
}

func (e *RagRetrievalEngine) applyRerank(subQuestion string, candidates []*klvo.Document, usedChannels []string) []*klvo.Document {
	if !e.properties.RerankEnabled || len(candidates) == 0 || e.rerankPostProcessor == nil {
		return candidates
	}

	markUsedChannel(usedChannels, "rerank", mu)
	result, err := e.rerankPostProcessor.Process(subQuestion, candidates)
	if err != nil {
		Warnf("重排序处理失败: subQuestion='%s', error=%v", subQuestion, err)
		return candidates
	}
	return result
}

func (e *RagRetrievalEngine) assignReferenceIds(evidenceList []*rvo.SubQuestionEvidence) {
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
	parts := make([]string, 0, len(channelResults))
	for _, result := range channelResults {
		count := 0
		if result != nil && result.Documents != nil {
			count = len(result.Documents)
		}
		parts = append(parts, fmt.Sprintf("%s=%d", result.ChannelName, count))
	}
	return strings.Join(parts, "，")
}

// buildChannelTraces 构建子问题渠道执行追踪
// 统计每个检索渠道的召回数量（原始结果）和接受数量（过滤后结果），用于追踪检索效果
func (e *RagRetrievalEngine) buildChannelTraces(rawResults, filteredResults []*RetrievalChannelResult) []*rvo.SubQuestionChannelTrace {
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
	return slice.Map(maputil.Keys(channelNames), func(index int, name string) *rvo.SubQuestionChannelTrace {
		return &rvo.SubQuestionChannelTrace{
			ChannelName:   name,
			RecalledCount: rawMap[name],
			AcceptedCount: filteredMap[name],
		}
	})
}

func Warnf(format string, args ...interface{}) {
	logx.Alert(fmt.Sprintf(format, args...))
}
