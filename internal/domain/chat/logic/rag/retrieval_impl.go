package rag

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/rerank"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/svc"
)

const rrfK = 60

// RetrievalImpl RAG 检索引擎实现
type RetrievalImpl struct {
	repo                      adapter.ChatRepository
	reranker                  rerank.Reranker
	channels                  []channel.Retrieval
	docGateway                adapter.DocumentGateway
	channelTimeout            time.Duration
	subQuestionTimeout        time.Duration
	minVectorSimilarity       float64
	keywordRelativeScoreFloor float64
	rerankScoreThreshold      float64
	candidateTopK             int
	parentEvidenceMaxChars    int
	rerankEnabled             bool
	finalTopK                 int
	vectorTopK                int
	keywordTopK               int
}

func NewRetrievalImpl(svcCtx *svc.ServiceContext, repo adapter.ChatRepository, reranker rerank.Reranker,
	channels []channel.Retrieval, docGateway adapter.DocumentGateway) *RetrievalImpl {
	return &RetrievalImpl{
		repo:                      repo,
		channels:                  channels,
		reranker:                  reranker,
		docGateway:                docGateway,
		subQuestionTimeout:        svcCtx.Config.Chat.Rag.SubQuestionTimeout,
		channelTimeout:            svcCtx.Config.Chat.Rag.ChannelTimeout,
		minVectorSimilarity:       svcCtx.Config.Chat.Rag.Vector.MinSimilarity,
		keywordRelativeScoreFloor: svcCtx.Config.Chat.Rag.Keyword.RelativeScoreFloor,
		rerankScoreThreshold:      svcCtx.Config.Chat.Rag.Rerank.ScoreThreshold,
		candidateTopK:             svcCtx.Config.Chat.Rag.CandidateTopK,
		parentEvidenceMaxChars:    svcCtx.Config.Chat.Rag.ParentEvidenceMaxChars,
		rerankEnabled:             svcCtx.Config.Chat.Rag.Rerank.Enabled,
		finalTopK:                 svcCtx.Config.Chat.Rag.FinalTopK,
		vectorTopK:                svcCtx.Config.Chat.Rag.Vector.TopK,
		keywordTopK:               svcCtx.Config.Chat.Rag.Keyword.TopK,
	}
}

var _ Retriever = (*RetrievalImpl)(nil)

// Retrieve RAG 检索入口，执行多子问题并行检索
func (e *RetrievalImpl) Retrieve(ctx context.Context, plan *vo.ConversationExecutionPlan) (*vo.RagRetrievalContext, error) {
	if err := plan.Validate(); err != nil {
		return nil, err
	}

	ragCtx := vo.NewRagRetrievalContext(plan.RetrievalQuestion)

	subQuestions := plan.RetrievalSubQuestions
	if len(subQuestions) == 0 {
		subQuestions = []string{plan.RetrievalQuestion}
	}

	evidenceList := e.retrieveSubQuestionParallel(ctx, ragCtx, subQuestions, plan)
	acceptedCount := slice.CountBy(evidenceList, func(index int, item *vo.SubQuestionEvidence) bool { return len(item.SourceDocuments) > 0 })

	logx.Infof("RAG 检索完成: retrievalQuestion='%s', originalSubQuestionCount=%d, acceptedSubQuestionCount=%d, notes=%v",
		plan.RetrievalQuestion, len(evidenceList), acceptedCount, ragCtx.RetrievalNotes())

	e.assignReferenceIds(evidenceList)
	ragCtx.SubQuestionEvidenceList = evidenceList

	return ragCtx, nil
}

// -------------------- 子问题并行检索 --------------------

// retrieveSubQuestionParallel 并行检索所有子问题，每个子问题独立执行完整的检索管线
//
// 管线流程：
//  1. 并行调用各通道检索 → 收集原始结果
//  2. 应用证据闸门过滤 → 构建通道追踪
//  3. RRF 融合 → 父块提升 → 重排序
//  4. 截取 finalTopK → 追加 GraphRAG canonical 观测
//  5. 记录通道/检索结果观测数据
func (e *RetrievalImpl) retrieveSubQuestionParallel(ctx context.Context, ragCtx *vo.RagRetrievalContext, subQuestions []string, plan *vo.ConversationExecutionPlan) []*vo.SubQuestionEvidence {
	timeoutCtx, cancel := context.WithTimeout(ctx, e.subQuestionTimeout)
	defer cancel()

	resultChan := make(chan *vo.SubQuestionEvidence, len(subQuestions))
	defer close(resultChan)

	for i, sq := range subQuestions {
		go func(subQuestionIndex int, subQuestion string) {
			start := time.Now()
			channelResults, err := e.retrieveChannelParallel(timeoutCtx, ragCtx, subQuestionIndex, subQuestion, plan)
			if err != nil {
				logx.Warnf("子问题检索失败: subQuestionIndex=%d, subQuestion='%v", subQuestionIndex, err)
				ragCtx.AddRetrievalNotef("子问题%d检索失败或超时，已自动忽略。", subQuestionIndex)
				resultChan <- &vo.SubQuestionEvidence{SubQuestionIndex: subQuestionIndex, SubQuestion: subQuestion}
				return
			}
			if len(channelResults) == 0 {
				ragCtx.AddRetrievalNotef("子问题%d没有可用的检索通道。", subQuestionIndex)
				resultChan <- &vo.SubQuestionEvidence{SubQuestionIndex: subQuestionIndex, SubQuestion: subQuestion}
				return
			}

			rawChannelResults := slice.Filter(channelResults, func(index int, result *vo.RetrievalChannelResult) bool {
				return len(result.Documents) > 0
			})
			filteredResults := slice.Map(rawChannelResults, func(index int, result *vo.RetrievalChannelResult) *vo.RetrievalChannelResult {
				return e.applyEvidenceGate(result)
			})

			channelTraces := e.buildChannelTraces(rawChannelResults, filteredResults)

			for _, r := range filteredResults {
				if len(r.Documents) > 0 {
					ragCtx.AddUsedChannel(r.ChannelName)
				}
			}

			fusedDocs := e.fuseByRRF(filteredResults)
			parentSearchDocs, err := e.elevateToParentChunks(timeoutCtx, fusedDocs, e.parentEvidenceMaxChars)
			if err != nil {
				logx.Warnf("父块提升失败: subQuestionIndex=%d, error=%v", subQuestionIndex, err)
				return
			}

			rerankedDocs := e.applyRerank(ctx, ragCtx, parentSearchDocs, subQuestion)

			finalTopK := min(e.finalTopK, len(rerankedDocs))
			finalDocs := rerankedDocs[:finalTopK]

			// GraphRAG canonical 观测
			e.appendGraphRagCanonicalNotes(ragCtx, subQuestionIndex, subQuestion, finalDocs)

			ragCtx.AddRetrievalNotef("子问题%d检索完成：%s，final=%d",
				subQuestionIndex, e.summarizeChannelResults(filteredResults), len(finalDocs))

			// 观测持久化状态
			var obsPersistence *vo.ObservationPersistence
			trace := vo.TraceFromCtx(ctx)
			if trace != nil {
				if err = e.recordChannelObservations(ctx, trace, subQuestionIndex, subQuestion, start, rawChannelResults, filteredResults, channelTraces); err != nil {
					logx.Warnf("记录通道观测数据失败: subQuestionIndex=%d, error=%v", subQuestionIndex, err)
				}
				obsPersistence = e.recordRetrievalResultObservations(ctx, trace, subQuestionIndex, subQuestion,
					rawChannelResults, filteredResults, fusedDocs, rerankedDocs, finalDocs)
			}

			resultChan <- &vo.SubQuestionEvidence{
				SubQuestionIndex:       subQuestionIndex,
				SubQuestion:            subQuestion,
				SourceDocuments:        finalDocs,
				ChannelTraces:          channelTraces,
				FusedCandidateCount:    len(fusedDocs),
				ParentCandidateCount:   len(parentSearchDocs),
				RerankedCandidateCount: len(rerankedDocs),
				ObservationPersistence: obsPersistence,
			}
		}(i+1, sq)
	}

	evidenceList := make([]*vo.SubQuestionEvidence, 0, len(subQuestions))
	for {
		select {
		case result := <-resultChan:
			evidenceList = append(evidenceList, result)
			if len(evidenceList) == len(subQuestions) {
				return evidenceList
			}
		case <-timeoutCtx.Done():
			return evidenceList
		}
	}
}

// -------------------- 通道检索 --------------------

// retrieveChannelParallel 并行检索单个子问题的所有通道
func (e *RetrievalImpl) retrieveChannelParallel(ctx context.Context, ragCtx *vo.RagRetrievalContext, subQuestionIndex int,
	subQuestion string, plan *vo.ConversationExecutionPlan) ([]*vo.RetrievalChannelResult, error) {
	// 创建带超时的上下文，超时时间为通道超时配置（保证单个通道异常不会阻塞整体）
	timeoutCtx, cancel := context.WithTimeout(ctx, e.channelTimeout)
	defer cancel()

	// 过滤出当前计划支持的通道（无通道直接返回空，让上游继续）
	channels := slice.Filter(e.channels, func(_ int, item channel.Retrieval) bool { return item.Supports(plan) })
	if len(channels) == 0 {
		return nil, nil
	}

	// 创建带缓冲的结果通道，容量 = 通道数量，避免 goroutine 写阻塞
	resultCh := make(chan *vo.RetrievalChannelResult, len(channels))
	defer close(resultCh)

	// 为每个通道启动一个 goroutine 并行执行检索
	for _, ch := range channels {
		go func(ch channel.Retrieval) {
			// 组装文档检索对象（传入子问题、执行计划、opK）
			topK := utils.Ternary(ch.ChannelName() == "vector", e.vectorTopK, e.candidateTopK)
			documentRetrieve := vo.NewDocumentRetrieve(subQuestion, plan, topK)
			// 调用 retrieveChannel（实际执行：加载文档元数据 → 调用通道检索 → 回填知识库信息）
			result, err := e.retrieveChannel(timeoutCtx, ch, documentRetrieve)
			if err != nil {
				// 失败/超时：仅告警并写入 RAG 上下文提示，返回空结果（自动降级）
				logx.Warnf("检索通道失败: subQuestionIndex=%d, subQuestion='%s', channel='%s', error=%v",
					subQuestionIndex, subQuestion, ch.ChannelName(), err)
				ragCtx.AddRetrievalNotef("子问题%d通道[%s]检索失败或超时，已自动降级。", subQuestionIndex, ch.ChannelName())
				result = &vo.RetrievalChannelResult{ChannelName: ch.ChannelName(), Documents: nil}
			}
			// 将结果写入结果通道（因 resultCh 带缓冲且容量等于通道数，此处不会阻塞）
			resultCh <- result
		}(ch)
	}

	// 主循环收集结果；一旦全部通道返回或上下文超时即退出
	channelResults := make([]*vo.RetrievalChannelResult, 0, len(channels))
	for {
		select {
		case result := <-resultCh:
			channelResults = append(channelResults, result)
			// 所有通道都返回时结束收集
			if len(channelResults) == len(channels) {
				return channelResults, nil
			}
		case <-timeoutCtx.Done():
			// 超时：返回已收集的部分结果 + DeadlineExceeded
			return channelResults, context.DeadlineExceeded
		}
	}
}

// retrieveChannel 调用单个检索通道，并将结果文档回填知识元信息
func (e *RetrievalImpl) retrieveChannel(ctx context.Context, ch channel.Retrieval, query *vo.DocumentRetrieve) (*vo.RetrievalChannelResult, error) {
	// 按查询中的文档元信息
	documents, err := e.docGateway.FetchRetrieveDocuments(ctx, query.DocumentIds...)
	if err != nil {
		return nil, err
	}
	knowledgeMap := utils.MapBy(documents, func(t *vo.DocumentMetadata) (int64, *vo.DocumentMetadata) {
		return t.DocumentId, t
	})

	result, err := ch.Retrieve(ctx, query)
	if err != nil {
		return nil, err
	}

	for _, doc := range result.Documents {
		doc.FillKnowledge(knowledgeMap[doc.DocumentId])
	}

	return result, nil
}

// -------------------- 父块提升 --------------------

// elevateToParentChunks 将子文档提升到父块级别，聚合出更完整的证据
func (e *RetrievalImpl) elevateToParentChunks(ctx context.Context, childDocuments []*vo.DocumentChunk, maxChars int) ([]*vo.DocumentChunk, error) {
	if len(childDocuments) == 0 {
		return nil, nil
	}

	childGroupsByParent := make(map[int64][]*vo.DocumentChunk, len(childDocuments))
	fallbackDocuments := make([]*vo.DocumentChunk, 0, len(childDocuments))
	parentBlockIds := make([]int64, 0, len(childDocuments))
	for _, childDocument := range childDocuments {
		parentChunkId := childDocument.ParentBlockId
		if parentChunkId == 0 {
			fallbackDocuments = append(fallbackDocuments, childDocument)
			continue
		}
		childGroupsByParent[parentChunkId] = append(childGroupsByParent[parentChunkId], childDocument)
		if _, exists := childGroupsByParent[parentChunkId]; exists {
			parentBlockIds = append(parentBlockIds, parentChunkId)
		}
	}

	if len(childGroupsByParent) == 0 {
		return fallbackDocuments, nil
	}

	parentChunks, err := e.docGateway.QueryParentChunks(ctx, parentBlockIds)
	if err != nil {
		return nil, err
	}
	parentChunkMap := utils.MapBy(parentChunks, func(item *vo.DocumentChunk) (string, *vo.DocumentChunk) {
		return item.ID, item
	})

	elevatedDocuments := make([]*vo.DocumentChunk, 0, len(childGroupsByParent)+len(fallbackDocuments))
	for parentId, children := range childGroupsByParent {
		parentBlock, ok := parentChunkMap[parentId]
		if !ok {
			elevatedDocuments = append(elevatedDocuments, children...)
			continue
		}
		elevatedDocuments = append(elevatedDocuments, e.buildParentEvidenceDocument(parentBlock, children, maxChars))
	}
	elevatedDocuments = append(elevatedDocuments, fallbackDocuments...)

	slices.SortFunc(elevatedDocuments, func(a, b *vo.DocumentChunk) int {
		if a.Score != b.Score {
			return int(b.Score - a.Score)
		} else if a.ParentBlockNo != b.ParentBlockNo {
			return a.ParentBlockNo - b.ParentBlockNo
		}
		return a.ChunkNo - b.ChunkNo
	})

	return elevatedDocuments, nil
}

// buildParentEvidenceDocument 构建父级证据文档
func (e *RetrievalImpl) buildParentEvidenceDocument(parentBlock *den.DocumentParentChunk, childDocuments []*vo.DocumentChunk, maxChars int) *vo.DocumentChunk {
	if parentBlock == nil || len(childDocuments) == 0 {
		return nil
	}

	bestChild := childDocuments[0]
	for i := 1; i < len(childDocuments); i++ {
		if bestChild.Score < childDocuments[i].Score {
			bestChild = childDocuments[i]
		}
	}

	channelMap := make(map[string]struct{})
	for _, childDocument := range childDocuments {
		channelMap[childDocument.Channel] = struct{}{}
	}
	channels := maputil.Keys(channelMap)

	supportCount := max(0, len(childDocuments)-1)
	supportWeight := min(0.36, float64(supportCount)*0.12)
	multiChannelWeight := utils.Ternary(len(channels) > 1, 0.10, 0.0)
	parentScore := bestChild.Score * (1.0 + supportWeight + multiChannelWeight)

	return &vo.DocumentChunk{
		ID:                fmt.Sprintf("parent-%d", parentBlock.ID),
		Content:           e.renderParentEvidenceText(parentBlock, childDocuments, maxChars),
		ParentBlockId:     parentBlock.ID,
		ParentBlockNo:     parentBlock.ParentNo,
		SectionPath:       parentBlock.SectionPath,
		StructureNodeId:   parentBlock.StructureNodeId,
		StructureNodeType: parentBlock.StructureNodeType,
		CanonicalPath:     parentBlock.CanonicalPath,
		ItemIndex:         parentBlock.ItemIndex,
		OriginalSnippet:   parentBlock.ParentText,
		IsElevated:        1,
		Score:             parentScore,
		Channel:           utils.Ternary(len(channels) > 1, "hybrid", channels[0]),
	}
}

// renderParentEvidenceText 渲染父级证据文本：[父块内容] + [命中子片段]
func (e *RetrievalImpl) renderParentEvidenceText(parentBlock *den.DocumentParentChunk, childDocuments []*vo.DocumentChunk, maxChars int) string {
	parentText := strutil.Trim(parentBlock.ParentText)

	if strutil.IsBlank(parentText) {
		if len(childDocuments) == 0 {
			return ""
		}
		return childDocuments[0].OriginalSnippet
	}

	var childSummaryBuilder strings.Builder
	for i, childDocument := range childDocuments {
		if i > 0 {
			childSummaryBuilder.WriteByte('\n')
		}
		childSummaryBuilder.WriteString("- child#")
		childSummaryBuilder.WriteString(strconv.Itoa(childDocument.ChunkNo))
		childSummaryBuilder.WriteString("：")
		childSummaryBuilder.WriteString(utils.ClipHead(childDocument.OriginalSnippet, 140))
	}

	var composed string
	if childSummaryBuilder.Len() > 0 {
		composed = fmt.Sprintf("[父块内容]\n%s\n\n[命中子片段]\n%s", parentText, childSummaryBuilder.String())
	} else {
		composed = fmt.Sprintf("[父块内容]\n%s", parentText)
	}

	return utils.ClipHead(composed, max(maxChars, 1))
}

// -------------------- 证据闸门 --------------------

// applyEvidenceGate 根据通道类型应用不同的分数过滤策略
func (e *RetrievalImpl) applyEvidenceGate(result *vo.RetrievalChannelResult) *vo.RetrievalChannelResult {
	if result == nil || len(result.Documents) == 0 {
		return result
	}

	var documents []*vo.DocumentChunk
	switch result.ChannelName {
	case enum.RetrievalChannelVector:
		documents = slice.Filter(result.Documents, func(index int, doc *vo.DocumentChunk) bool {
			return doc.Score >= e.minVectorSimilarity
		})
	case enum.RetrievalChannelKeyword:
		maxScore := slices.MaxFunc(result.Documents, func(doc1, doc2 *vo.DocumentChunk) int { return int(doc1.Score - doc2.Score) }).Score
		documents = slice.Filter(result.Documents, func(index int, doc *vo.DocumentChunk) bool {
			return doc.Score >= (e.keywordRelativeScoreFloor * maxScore)
		})
	default:
		documents = result.Documents
	}

	return &vo.RetrievalChannelResult{
		ChannelName: result.ChannelName,
		Documents:   documents,
	}
}

// -------------------- RRF 融合 --------------------

type candidateHolder struct {
	document *vo.DocumentChunk
	score    float64
	channels map[string]struct{}
}

// fuseByRRF 基于 RRF 算法融合多通道候选结果
func (e *RetrievalImpl) fuseByRRF(channelResults []*vo.RetrievalChannelResult) []*vo.DocumentChunk {
	var holders []*candidateHolder

	for _, channelResult := range channelResults {
		holders = e.accumulateRRF(channelResult)
	}

	result := make([]*vo.DocumentChunk, 0, len(holders))
	stream.FromSlice(holders).
		Sorted(func(a, b *candidateHolder) bool { return a.score > b.score }).
		Limit(e.candidateTopK).
		ForEach(func(holder *candidateHolder) {
			holder.document.RRFScore = holder.score
			holder.document.Channel = utils.Ternary(len(holder.channels) > 1, enum.RetrievalChannelHybrid, maputil.Keys(holder.channels)[0])
			result = append(result, holder.document)
		})
	return result
}

// accumulateRRF 计算文档的 RRF 分数
func (e *RetrievalImpl) accumulateRRF(channelResult *vo.RetrievalChannelResult) []*candidateHolder {
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

// -------------------- 重排序 --------------------

// applyRerank 应用重排序（如果启用）
func (e *RetrievalImpl) applyRerank(ctx context.Context, ragCtx *vo.RagRetrievalContext, candidates []*vo.DocumentChunk, subQuestion string) []*vo.DocumentChunk {
	if !e.rerankEnabled || len(candidates) == 0 || e.reranker == nil {
		return candidates
	}

	ragCtx.AddUsedChannel(enum.RetrievalChannelRerank)
	result, err := e.reranker.Process(ctx, subQuestion, candidates)
	if err != nil {
		logx.Warnf("重排序处理失败: subQuestion='%s', error=%v", subQuestion, err)
		return candidates
	}
	return result
}

// -------------------- 引用 ID 分配 --------------------

// assignReferenceIds 为检索证据分配引用 ID，并回填知识库元数据
func (e *RetrievalImpl) assignReferenceIds(evidenceList []*vo.SubQuestionEvidence) {
	referenceNumber := 1
	assignedIds := make(map[string]string)

	for _, evidence := range evidenceList {
		references := make([]*vo.SearchReference, 0, len(evidence.SourceDocuments))
		for _, doc := range evidence.SourceDocuments {
			ref := vo.NewSearchReference(doc, evidence.SubQuestionIndex, 0, evidence.SubQuestion)
			uniqueKey := ref.UniqueKey()

			assignedId, ok := assignedIds[uniqueKey]
			if !ok {
				assignedId = fmt.Sprintf("%d", referenceNumber)
				assignedIds[uniqueKey] = assignedId
				referenceNumber++
			}
			ref.ReferenceId = assignedId
			references = append(references, ref)
		}
		evidence.References = references
	}
}

// -------------------- 通道追踪 --------------------

// buildChannelTraces 构建子问题渠道执行追踪
func (e *RetrievalImpl) buildChannelTraces(rawResults, filteredResults []*vo.RetrievalChannelResult) []*vo.SubQuestionChannelTrace {
	if len(rawResults) == 0 && len(filteredResults) == 0 {
		return nil
	}

	rawMap := make(map[string]int)
	filteredMap := make(map[string]int)
	channelNames := make(map[string]struct{})

	slice.ForEach(rawResults, func(index int, r *vo.RetrievalChannelResult) {
		rawMap[r.ChannelName] = len(r.Documents)
		channelNames[r.ChannelName] = struct{}{}
	})
	slice.ForEach(filteredResults, func(index int, r *vo.RetrievalChannelResult) {
		filteredMap[r.ChannelName] = len(r.Documents)
		channelNames[r.ChannelName] = struct{}{}
	})

	return slice.Map(maputil.Keys(channelNames), func(index int, name string) *vo.SubQuestionChannelTrace {
		return &vo.SubQuestionChannelTrace{
			ChannelName:   name,
			RecalledCount: rawMap[name],
			AcceptedCount: filteredMap[name],
		}
	})
}

// summarizeChannelResults 摘要每个检索渠道的文档数量
func (e *RetrievalImpl) summarizeChannelResults(channelResults []*vo.RetrievalChannelResult) string {
	if len(channelResults) == 0 {
		return "没有启用任何检索通道"
	}
	parts := slice.Map(channelResults, func(_ int, result *vo.RetrievalChannelResult) string {
		return fmt.Sprintf("%s=%d", result.ChannelName, len(result.Documents))
	})
	return strings.Join(parts, "，")
}

// -------------------- 观测数据记录 --------------------

// recordChannelObservations 记录渠道执行观测数据
func (e *RetrievalImpl) recordChannelObservations(ctx context.Context, trace *vo.ConversationTrace, subQuestionIndex int, subQuestion string,
	start time.Time, rawResults, filteredResults []*vo.RetrievalChannelResult, channelTraces []*vo.SubQuestionChannelTrace) error {
	if len(rawResults) == 0 {
		return nil
	}

	end := time.Now()
	executions := make([]*vo.ChatChannelExecution, 0, len(rawResults))

	filteredResultsMap := utils.MapBy(filteredResults, func(r *vo.RetrievalChannelResult) (string, *vo.RetrievalChannelResult) {
		return r.ChannelName, r
	})
	channelTracesMap := utils.MapBy(channelTraces, func(t *vo.SubQuestionChannelTrace) (string, *vo.SubQuestionChannelTrace) {
		return t.ChannelName, t
	})

	for _, rawResult := range rawResults {
		channelName := rawResult.ChannelName
		execution := &vo.ChatChannelExecution{
			ConversationId:   trace.ConversationId(),
			ExchangeId:       trace.ExchangeId(),
			TraceId:          trace.TraceId(),
			SubQuestionIndex: subQuestionIndex,
			SubQuestion:      subQuestion,
			ChannelType:      channelName,
			StartTime:        start,
			EndTime:          end,
			DurationMs:       end.Sub(start).Milliseconds(),
			ExecutionState:   1,
			RecalledCount:    len(rawResult.Documents),
		}

		if filteredResult, ok := filteredResultsMap[channelName]; ok {
			execution.AcceptedCount = len(filteredResult.Documents)
		}
		if trace, ok := channelTracesMap[channelName]; ok {
			execution.FinalSelectedCount = trace.AcceptedCount
		}
		execution.SetScores(rawResult.Documents)
		executions = append(executions, execution)
	}

	return e.repo.InsertChannelExecutions(ctx, executions)
}

// recordRetrievalResultObservations 记录检索结果观测数据，返回观测持久化状态
//
// 为每个渠道的每个原始文档生成一条 ChatRetrievalResult 记录，追踪全流程中每篇文档的状态与原因。
// 与 Java 版本 RagRetrievalEngine 中的 projectAndPersistRetrievalResultObservations 对应。
func (e *RetrievalImpl) recordRetrievalResultObservations(ctx context.Context, trace *vo.ConversationTrace, subQuestionIndex int, subQuestion string,
	rawResults, filteredResults []*vo.RetrievalChannelResult, fusedDocs, rerankedDocs, finalDocs []*vo.DocumentChunk) *vo.ObservationPersistence {
	if len(rawResults) == 0 {
		return nil
	}

	expectedCandidateCount := e.rawCandidateCount(rawResults)
	obsPersistence := &vo.ObservationPersistence{
		ExpectedCandidateCount: expectedCandidateCount,
	}

	results, err := e.projectRetrievalResults(subQuestionIndex, subQuestion, trace,
		rawResults, filteredResults, fusedDocs, rerankedDocs, finalDocs)
	if err != nil {
		logx.Warnf("投影检索候选观测数据失败: subQuestionIndex=%d, expectedCandidateCount=%d, error=%v",
			subQuestionIndex, expectedCandidateCount, err)
		// 观测投影失败，返回失败的持久化状态
		return obsPersistence
	}

	if err = e.repo.InsertRetrievalResults(ctx, results); err != nil {
		logx.Warnf("写入检索结果观测数据失败: subQuestionIndex=%d, error=%v", subQuestionIndex, err)
		return obsPersistence
	}

	obsPersistence.PersistedCandidateCount = len(results)
	return obsPersistence
}

// projectRetrievalResults 将原始/过滤/融合/重排/最终文档投影为 ChatRetrievalResult 列表
func (e *RetrievalImpl) projectRetrievalResults(subQuestionIndex int, subQuestion string, trace *vo.ConversationTrace,
	rawResults, filteredResults []*vo.RetrievalChannelResult, fusedDocs, rerankedDocs, finalDocs []*vo.DocumentChunk) ([]*vo.ChatRetrievalResult, error) {
	// 构建最终文档 FinalRank 映射（按 ParentBlockId）
	finalRankMap := make(map[int64]int)
	for i, doc := range finalDocs {
		finalRankMap[doc.ParentBlockId] = i + 1
	}

	// 按 RRFScore 降序排序 fusedDocs，构建 RrfRank 映射
	sort.Slice(fusedDocs, func(i, j int) bool {
		return fusedDocs[i].RRFScore > fusedDocs[j].RRFScore
	})
	rrfRankMap := make(map[string]int)
	rrfScoreMap := make(map[string]float64)
	for i, doc := range fusedDocs {
		rrfRankMap[doc.ID] = i + 1
		rrfScoreMap[doc.ID] = doc.RRFScore
	}

	rerankScoreMap := make(map[int64]float64)
	for _, doc := range rerankedDocs {
		rerankScoreMap[doc.ParentBlockId] = doc.RerankScore
	}

	// 构建"通过闸门"的文档 ID 集合（按渠道名分组）
	gatePassedSet := make(map[string]map[string]int)
	for _, fr := range filteredResults {
		gatePassedSet[fr.ChannelName] = make(map[string]int)
		for _, doc := range fr.Documents {
			gatePassedSet[fr.ChannelName][doc.ID] = 1
		}
	}

	results := make([]*vo.ChatRetrievalResult, 0, e.rawCandidateCount(rawResults))
	for _, rawResult := range rawResults {
		channelName := rawResult.ChannelName
		for i, doc := range rawResult.Documents {
			view := &vo.ChatRetrievalResult{
				ConversationId:   trace.ConversationId(),
				ExchangeId:       trace.ExchangeId(),
				TraceId:          trace.TraceId(),
				SubQuestionIndex: subQuestionIndex,
				SubQuestion:      subQuestion,
				ChannelType:      channelName,
				ChannelRank:      i + 1,
				RrfRank:          rrfRankMap[doc.ID],
				RrfScore:         rrfScoreMap[doc.ID],
				RerankScore:      rerankScoreMap[doc.ParentBlockId],
				GatePassed:       gatePassedSet[channelName][doc.ID],
			}
			view.SetDocumentInfo(doc)

			// 判定是否被选入最终 Prompt
			if view.GatePassed == 0 {
				view.SelectionReason = e.resolveGateFilteredReason(channelName)
			} else if rank, ok := finalRankMap[doc.ParentBlockId]; ok {
				view.IsSelected = 1
				view.FinalRank = rank
				view.SelectionReason = "已选入最终 Prompt"
			} else {
				view.SelectionReason = fmt.Sprintf("超出 finalTopK 限制（topK=%d）", e.finalTopK)
			}

			results = append(results, view)
		}
	}

	return results, nil
}

// rawCandidateCount 统计原始结果的候选文档总数
func (e *RetrievalImpl) rawCandidateCount(rawResults []*vo.RetrievalChannelResult) int {
	count := 0
	for _, r := range rawResults {
		count += len(r.Documents)
	}
	return count
}

// resolveGateFilteredReason 根据渠道类型返回闸门过滤原因
func (e *RetrievalImpl) resolveGateFilteredReason(channelName string) string {
	switch channelName {
	case enum.RetrievalChannelVector:
		return fmt.Sprintf("向量闸门过滤：分数 < 阈值 %.4f", e.minVectorSimilarity)
	case enum.RetrievalChannelKeyword:
		return fmt.Sprintf("关键词闸门过滤：分数低于相对阈值（floor=%.2f）", e.keywordRelativeScoreFloor)
	default:
		return "闸门过滤"
	}
}

// -------------------- GraphRAG Canonical 观测 --------------------

// graphRagCanonicalObservation GraphRAG canonical 观测记录
type graphRagCanonicalObservation struct {
	text          string
	priority      float64
	originalIndex int
}

// appendGraphRagCanonicalNotes 为最终文档中的 GraphRAG 文档生成 canonical 观测笔记
//
// 对应 Java 版本 RagRetrievalEngine.appendGraphRagCanonicalNotes。
// 检测最终文档中的 GraphRAG 元数据，生成结构化的观测摘要并追加到 RetrievalNotes。
func (e *RetrievalImpl) appendGraphRagCanonicalNotes(ragCtx *vo.RagRetrievalContext, subQuestionIndex int, subQuestion string, finalDocs []*vo.DocumentChunk) {
	if len(finalDocs) == 0 {
		return
	}

	var observations []graphRagCanonicalObservation
	for idx, doc := range finalDocs {
		if !e.isGraphRagDocument(doc) {
			continue
		}
		obs := e.buildGraphRagObservation(doc, idx)
		observations = append(observations, obs)
	}

	if len(observations) == 0 {
		return
	}

	// 按优先级降序、原始索引升序排序，取前 4 条
	sort.Slice(observations, func(i, j int) bool {
		if observations[i].priority != observations[j].priority {
			return observations[i].priority > observations[j].priority
		}
		return observations[i].originalIndex < observations[j].originalIndex
	})
	limit := min(4, len(observations))
	parts := make([]string, limit)
	for i := 0; i < limit; i++ {
		parts[i] = observations[i].text
	}
	summary := strings.Join(parts, "；")
	ragCtx.AddRetrievalNotef("子问题%d GraphRAG canonical 观测：%s", subQuestionIndex, summary)
	logx.Infof("GraphRAG canonical 观测: subQuestionIndex=%d, subQuestion='%s', observations=%s",
		subQuestionIndex, subQuestion, summary)
}

// isGraphRagDocument 判断文档是否为 GraphRAG 来源
//
// 通过检视文档的渠道、来源类型和知识范围编码来判断。
// GraphRAG 文档通常由图谱检索通道产生，携带 KG 相关的元数据。
func (e *RetrievalImpl) isGraphRagDocument(doc *vo.DocumentChunk) bool {
	if doc == nil {
		return false
	}
	return doc.SourceType == "GRAPH_RAG" ||
		doc.Channel == "graph_rag" ||
		doc.KnowledgeScopeCode == "GRAPH_RAG" ||
		strings.Contains(doc.Channel, "graph_rag") ||
		strings.Contains(doc.SourceType, "GRAPH_RAG")
}

// buildGraphRagObservation 为单个 GraphRAG 文档构建观测记录
func (e *RetrievalImpl) buildGraphRagObservation(doc *vo.DocumentChunk, originalIndex int) graphRagCanonicalObservation {
	var builder strings.Builder

	// 构建 canonical 标识
	canonicalName := e.firstNonBlank(doc.DocumentName, doc.Title, "unknown")
	builder.WriteString(canonicalName)

	// 补充文档统计信息
	score := e.finalDocumentScore(doc)
	priority := math.Max(0, score)
	priority -= float64(originalIndex) * 0.0001

	text := builder.String()
	return graphRagCanonicalObservation{
		text:          text,
		priority:      priority,
		originalIndex: originalIndex,
	}
}

// finalDocumentScore 获取文档的最终分数
func (e *RetrievalImpl) finalDocumentScore(doc *vo.DocumentChunk) float64 {
	if doc == nil {
		return 0
	}
	score := e.resolveScore(doc)
	if score != 0 {
		return score
	}
	return doc.Score
}

// resolveScore 解析文档的分数，优先使用元数据中的分数
func (e *RetrievalImpl) resolveScore(doc *vo.DocumentChunk) float64 {
	if doc == nil {
		return 0
	}
	return doc.RRFScore
}

// -------------------- 元数据工具方法 --------------------

// isMeaningfulMetadataValue 判断元数据值是否有意义
func (e *RetrievalImpl) isMeaningfulMetadataValue(value string) bool {
	return strutil.IsNotBlank(value)
}

// firstNonBlank 返回第一个非空字符串
func (e *RetrievalImpl) firstNonBlank(values ...string) string {
	for _, v := range values {
		if strutil.IsNotBlank(v) {
			return v
		}
	}
	return ""
}

// numericMetadataValue 将元数据值解析为 float64
func (e *RetrievalImpl) numericMetadataValue(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		parsed, _ := strconv.ParseFloat(v, 64)
		return parsed
	default:
		return 0
	}
}

// safeText 安全获取文本值
func (e *RetrievalImpl) safeText(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

// booleanMetadataValue 将元数据值解析为布尔值
func (e *RetrievalImpl) booleanMetadataValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		parsed, _ := strconv.ParseBool(v)
		return parsed
	default:
		return false
	}
}

// integerMetadataValue 将元数据值解析为 int
func (e *RetrievalImpl) integerMetadataValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		parsed, _ := strconv.Atoi(v)
		return parsed
	default:
		return 0
	}
}

// joinSections 拼接多个段，自动跳过空段
func (e *RetrievalImpl) joinSections(sections ...string) string {
	var builder strings.Builder
	for _, section := range sections {
		if strutil.IsBlank(section) {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(section)
	}
	return builder.String()
}
