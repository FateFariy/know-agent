package rag

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag/channel"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	doclog "github.com/swiftbit/know-agent/internal/domain/document/logic"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const rrfK = 60

type RetrievalImpl struct {
	channels                  []channel.RetrievalChannel
	documentLogic             doclog.LifecycleLogic
	channelTimeout            time.Duration
	subQuestionTimeout        time.Duration
	minVectorSimilarity       float64
	keywordRelativeScoreFloor float64
	candidateTopK             int
	parentEvidenceMaxChars    int
	rerankEnabled             bool
	finalTopK                 int
	vectorTopK                int
	keywordTopK               int
}

func NewRetrievalImpl(svcCtx *svc.ServiceContext, channels []channel.RetrievalChannel, documentLogic doclog.LifecycleLogic) *RetrievalImpl {
	return &RetrievalImpl{
		channels:                  channels,
		documentLogic:             documentLogic,
		subQuestionTimeout:        svcCtx.Config.Chat.Rag.SubQuestionTimeout,
		channelTimeout:            svcCtx.Config.Chat.Rag.ChannelTimeout,
		minVectorSimilarity:       svcCtx.Config.Chat.Rag.MinVectorSimilarity,
		keywordRelativeScoreFloor: svcCtx.Config.Chat.Rag.KeywordRelativeScoreFloor,
		candidateTopK:             svcCtx.Config.Chat.Rag.CandidateTopK,
		parentEvidenceMaxChars:    svcCtx.Config.Chat.Rag.ParentEvidenceMaxChars,
		rerankEnabled:             svcCtx.Config.Chat.Rag.RerankEnabled,
		finalTopK:                 svcCtx.Config.Chat.Rag.FinalTopK,
		vectorTopK:                svcCtx.Config.Chat.Rag.VectorTopK,
		keywordTopK:               svcCtx.Config.Chat.Rag.KeywordTopK,
	}
}

var _ logic.RagRetriever = (*RetrievalImpl)(nil)

func (e *RetrievalImpl) Retrieve(ctx context.Context, plan *vo.ConversationExecutionPlan, trace *vo.ConversationTrace) (*vo.RagRetrievalContext, error) {
	ragCtx := vo.NewRagRetrievalContext(plan.RetrievalQuestion)

	subQuestions := plan.RetrievalSubQuestions
	if len(subQuestions) == 0 {
		subQuestions = []string{plan.RetrievalQuestion}
	}

	evidenceList := e.retrieveSubQuestionParallel(ctx, ragCtx, subQuestions, plan, trace)
	acceptedCount := slice.CountBy(evidenceList, func(index int, item *vo.SubQuestionEvidence) bool { return len(item.Documents) > 0 })

	logx.Infof("RAG 检索完成: retrievalQuestion='%s', originalSubQuestionCount=%d, acceptedSubQuestionCount=%d, notes=%v",
		plan.RetrievalQuestion, len(evidenceList), acceptedCount, ragCtx.RetrievalNotes())

	e.assignReferenceIds(evidenceList)
	ragCtx.SubQuestionEvidenceList = evidenceList

	return ragCtx, nil
}

func (e *RetrievalImpl) retrieveSubQuestionParallel(ctx context.Context, ragCtx *vo.RagRetrievalContext, subQuestions []string,
	plan *vo.ConversationExecutionPlan, trace *vo.ConversationTrace) []*vo.SubQuestionEvidence {
	timeoutCtx, cancel := context.WithTimeout(ctx, e.subQuestionTimeout)
	defer cancel()

	resultChan := make(chan *vo.SubQuestionEvidence, len(subQuestions))
	defer close(resultChan)

	// todo 线程池改造
	for i, sq := range subQuestions {
		go func(subQuestionIndex int, subQuestion string) {
			channelResults, err := e.retrieveChannelParallel(timeoutCtx, ragCtx, subQuestionIndex, subQuestion, plan)
			if err != nil {
				Warnf("子问题检索失败: subQuestionIndex=%d, subQuestion='%v", subQuestionIndex, err)
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
			parentSearchDocs, err := e.elevateToParentBlocks(timeoutCtx, fusedDocs, e.parentEvidenceMaxChars)
			if err != nil {
				Warnf("父块提升失败: subQuestionIndex=%d, error=%v", subQuestionIndex, err)
				return
			}

			rerankedCandidates := e.applyRerank(ragCtx, parentSearchDocs, subQuestion)

			finalTopK := min(e.finalTopK, len(rerankedCandidates))
			finalDocuments := rerankedCandidates[:finalTopK]

			ragCtx.AddRetrievalNotef("子问题%d检索完成：%s，final=%d",
				subQuestionIndex, e.summarizeChannelResults(filteredResults), len(finalDocuments))

			// 记录观测数据
			if trace != nil {
				if err = e.recordChannelObservations(trace, subQuestionIndex, subQuestion, rawChannelResults, filteredResults, channelTraces); err != nil {
					Warnf("记录通道观测数据失败: subQuestionIndex=%d, error=%v", subQuestionIndex, err)
				}
				if err = e.recordRetrievalResultObservations(trace, subQuestionIndex, subQuestion, rawChannelResults, filteredResults, finalDocuments); err != nil {
					Warnf("记录检索结果观测数据失败: subQuestionIndex=%d, error=%v", subQuestionIndex, err)
				}
			}

			resultChan <- &vo.SubQuestionEvidence{
				SubQuestionIndex:       subQuestionIndex,
				SubQuestion:            subQuestion,
				Documents:              finalDocuments,
				ChannelTraces:          channelTraces,
				FusedCandidateCount:    len(fusedDocs),
				ParentCandidateCount:   len(parentSearchDocs),
				RerankedCandidateCount: len(rerankedCandidates),
			}
		}(i+1, sq)
	}

	// 收集所有子问题的结果
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

// retrieveChannelParallel 并行检索单个子问题的所有通道。
//
// 执行流程：
//  1. 创建带超时的上下文（channelTimeout），用于防止单个通道阻塞整个检索
//  2. 过滤出当前计划支持的通道（Supports），无通道时直接返回空
//  3. 为每个通道启动一个 goroutine 执行检索；失败/超时仅告警并返回空文档（即"自动降级"）
//  4. 主循环通过 select 收集结果或在超时退出时返回已收集的部分结果
func (e *RetrievalImpl) retrieveChannelParallel(ctx context.Context, ragCtx *vo.RagRetrievalContext, subQuestionIndex int,
	subQuestion string, plan *vo.ConversationExecutionPlan) ([]*vo.RetrievalChannelResult, error) {
	// 创建带超时的上下文，超时时间为通道超时配置（保证单个通道异常不会阻塞整体）
	timeoutCtx, cancel := context.WithTimeout(ctx, e.channelTimeout)
	defer cancel()

	// 过滤出当前计划支持的通道（无通道直接返回空，让上游继续）
	channels := slice.Filter(e.channels, func(_ int, item channel.RetrievalChannel) bool { return item.Supports(plan) })
	if len(channels) == 0 {
		return nil, nil
	}

	// 创建带缓冲的结果通道，容量 = 通道数量，避免 goroutine 写阻塞
	resultCh := make(chan *vo.RetrievalChannelResult, len(channels))
	defer close(resultCh)

	// 为每个通道启动一个 goroutine 并行执行检索
	for _, ch := range channels {
		go func(ch channel.RetrievalChannel) {
			// 组装文档检索对象（传入子问题、执行计划、向量 topK）
			documentRetrieve := vo.NewDocumentRetrieve(subQuestion, plan, e.vectorTopK)
			// 调用 retrieveChannel（实际执行：加载文档元数据 → 调用通道检索 → 回填知识库信息）
			result, err := e.retrieveChannel(timeoutCtx, ch, documentRetrieve)
			if err != nil {
				// 失败/超时：仅告警并写入 RAG 上下文提示，返回空结果（自动降级）
				Warnf("检索通道失败: subQuestionIndex=%d, subQuestion='%s', channel='%s', error=%v",
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

// retrieveChannel 调用单个检索通道，并将通道返回的文档回填知识元信息。
//
// 执行流程：
//  1. 加载 DocumentIds 对应的全部可检索文档，并按 DocumentId 索引为 map
//  2. 调用通道的 Retrieve 接口执行实际检索
//  3. 将返回的每个文档根据 DocumentId 从 map 中回填知识信息（名称、范围、标签等）
func (e *RetrievalImpl) retrieveChannel(ctx context.Context, ch channel.RetrievalChannel, query *vo.DocumentRetrieve) (*vo.RetrievalChannelResult, error) {
	// 按查询中的文档元信息
	documents, err := e.documentLogic.ListRetrievableDocuments(ctx, query.DocumentIds...)
	if err != nil {
		return nil, err
	}
	knowledgeMap := utils.SliceToMapBy(documents, func(t *dvo.KnowledgeDocument) (int64, *dvo.KnowledgeDocument) {
		return t.DocumentId, t
	})

	// 调用通道执行实际检索（向量 / 关键词 / 混合等，由通道实现决定）
	result, err := ch.Retrieve(ctx, query)
	if err != nil {
		return nil, err
	}

	// 对返回的每个文档，根据 DocumentId 回填知识库元信息
	for _, document := range result.Documents {
		document.FillKnowledge(knowledgeMap[document.DocumentId])
	}

	return result, nil
}

// elevateToParentBlocks 将子文档提升到父块级别，聚合出更完整的证据
// 流程：按 parentBlockId 分组 → 查询父块 → 聚合分数/通道 → 按分数排序
func (e *RetrievalImpl) elevateToParentBlocks(ctx context.Context, childDocuments []*vo.DocumentChunk, maxChars int) ([]*vo.DocumentChunk, error) {
	if len(childDocuments) == 0 {
		return nil, nil
	}

	// 按 parentBlockId 分组，并收集无法被归类的 childDocument 作为 fallback
	childGroupsByParent := make(map[int64][]*vo.DocumentChunk, len(childDocuments))
	fallbackDocuments := make([]*vo.DocumentChunk, 0, len(childDocuments))
	parentBlockIds := make([]int64, 0, len(childDocuments))
	for _, childDocument := range childDocuments {
		parentBlockId := childDocument.ParentBlockId
		if parentBlockId == 0 {
			fallbackDocuments = append(fallbackDocuments, childDocument)
			continue
		}
		childGroupsByParent[parentBlockId] = append(childGroupsByParent[parentBlockId], childDocument)
		if _, exists := childGroupsByParent[parentBlockId]; exists {
			parentBlockIds = append(parentBlockIds, parentBlockId)
		}
	}

	if len(childGroupsByParent) == 0 {
		return fallbackDocuments, nil
	}

	// 查询父块
	parentBlocks, err := e.documentLogic.QueryParentBlocks(ctx, parentBlockIds)
	if err != nil {
		return nil, err
	}
	parentBlockMap := utils.SliceToMapBy(parentBlocks, func(item *entity.DocumentParentBlock) (int64, *entity.DocumentParentBlock) {
		return item.ID, item
	})

	// 构建父级证据文档，或当父块未找到时直接保留子文档
	elevatedDocuments := make([]*vo.DocumentChunk, 0, len(childGroupsByParent)+len(fallbackDocuments))
	for parentId, children := range childGroupsByParent {
		parentBlock, ok := parentBlockMap[parentId]
		if !ok {
			elevatedDocuments = append(elevatedDocuments, children...)
			continue
		}
		elevatedDocuments = append(elevatedDocuments, e.buildParentEvidenceDocument(parentBlock, children, maxChars))
	}
	elevatedDocuments = append(elevatedDocuments, fallbackDocuments...)

	// 排序（分数降序 → 父块编号升序 → chunkNo 升序）
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
func (e *RetrievalImpl) buildParentEvidenceDocument(parentBlock *entity.DocumentParentBlock, childDocuments []*vo.DocumentChunk, maxChars int) *vo.DocumentChunk {
	if parentBlock == nil || len(childDocuments) == 0 {
		return nil
	}

	// 选出 score 最高的子文档，作为元数据的基础
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

	// 计算父级证据分数
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
		Score:             parentScore,
		Channel:           utils.Ternary(len(channels) > 1, "hybrid", channels[0]),
	}
}

// applyEvidenceGate 根据通道类型应用不同的分数过滤策略，过滤掉置信度不足的文档
func (e *RetrievalImpl) applyEvidenceGate(result *vo.RetrievalChannelResult) *vo.RetrievalChannelResult {
	if result == nil || len(result.Documents) == 0 {
		return result
	}

	var documents []*vo.DocumentChunk
	switch result.ChannelName {
	case vo.RetrievalChannelVector:
		// 向量通道：使用绝对相似度阈值过滤
		documents = slice.Filter(result.Documents, func(index int, doc *vo.DocumentChunk) bool {
			return doc.Score >= e.minVectorSimilarity
		})
	case vo.RetrievalChannelKeyword:
		// 关键词通道：使用相对分数阈值过滤（相对于最高分）
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

type candidateHolder struct {
	document *vo.DocumentChunk
	score    float64
	channels map[string]struct{}
}

// fuseByRRF 融合多个通道的候选结果（基于RRF算法）
// RRF(Reciprocal Rank Fusion)通过合并各通道的排名信息，计算综合分数，实现多通道结果融合
func (e *RetrievalImpl) fuseByRRF(channelResults []*vo.RetrievalChannelResult) []*vo.DocumentChunk {
	var holders []*candidateHolder

	// 遍历所有通道结果，累积计算RRF分数
	for _, channelResult := range channelResults {
		holders = e.accumulateRRF(channelResult)
	}

	// 按RRF分数降序排序，取前N个文档
	result := make([]*vo.DocumentChunk, 0, len(holders))
	stream.FromSlice(holders).
		Sorted(func(a, b *candidateHolder) bool { return a.score > b.score }).
		Limit(e.candidateTopK).
		ForEach(func(holder *candidateHolder) {
			// 填充文档元数据（分数、通道来源）
			holder.document.Score = holder.score
			holder.document.RRFScore = holder.score
			holder.document.Channel = utils.Ternary(len(holder.channels) > 1, vo.RetrievalChannelHybrid, maputil.Keys(holder.channels)[0])
			result = append(result, holder.document)
		})
	return result
}

// accumulateRRF 计算文档的RRF分数
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

func (e *RetrievalImpl) applyRerank(ragCtx *vo.RagRetrievalContext, candidates []*vo.DocumentChunk, subQuestion string) []*vo.DocumentChunk {
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

// assignReferenceIds 为检索证据分配引用ID
func (e *RetrievalImpl) assignReferenceIds(evidenceList []*vo.SubQuestionEvidence) {
	referenceNumber := 1
	assignedIDs := make(map[string]string)

	for _, evidence := range evidenceList {
		references := make([]*vo.SearchReference, 0, len(evidence.Documents))
		for _, doc := range evidence.Documents {
			ref := vo.NewSearchReference(doc, evidence.SubQuestionIndex, 0, evidence.SubQuestion)
			uniqueKey := ref.UniqueKey()

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

func (e *RetrievalImpl) summarizeChannelResults(channelResults []*vo.RetrievalChannelResult) string {
	if len(channelResults) == 0 {
		return "没有启用任何检索通道"
	}
	parts := slice.Map(channelResults, func(_ int, result *vo.RetrievalChannelResult) string {
		return fmt.Sprintf("%s=%d", result.ChannelName, len(result.Documents))
	})
	return strings.Join(parts, "，")
}

// buildChannelTraces 构建子问题渠道执行追踪
// 统计每个检索渠道的召回数量（原始结果）和接受数量（过滤后结果），用于追踪检索效果
func (e *RetrievalImpl) buildChannelTraces(rawResults, filteredResults []*vo.RetrievalChannelResult) []*vo.SubQuestionChannelTrace {
	if len(rawResults) == 0 && len(filteredResults) == 0 {
		return nil
	}

	// 初始化统计映射，key为渠道名称，value为文档数量
	rawMap := make(map[string]int)
	filteredMap := make(map[string]int)
	channelNames := make(map[string]struct{})

	// 统计原始检索结果中各渠道的文档数量
	slice.ForEach(rawResults, func(index int, r *vo.RetrievalChannelResult) {
		rawMap[r.ChannelName] = len(r.Documents)
		channelNames[r.ChannelName] = struct{}{}
	})

	// 统计过滤后结果中各渠道的文档数量
	slice.ForEach(filteredResults, func(index int, r *vo.RetrievalChannelResult) {
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

// recordChannelObservations 记录渠道执行观测数据，包括召回数量、接受数量、分数等
func (e *RetrievalImpl) recordChannelObservations(trace *vo.ConversationTrace, subQuestionIndex int, subQuestion string,
	rawResults, filteredResults []*vo.RetrievalChannelResult, channelTraces []*vo.SubQuestionChannelTrace) error {
	if len(rawResults) == 0 {
		return nil
	}

	executions := make([]*vo.ChannelExecutionView, 0, len(rawResults))
	filteredResultsMap := utils.SliceToMapBy(filteredResults, func(r *vo.RetrievalChannelResult) (string, *vo.RetrievalChannelResult) {
		return r.ChannelName, r
	})
	channelTracesMap := utils.SliceToMapBy(channelTraces, func(t *vo.SubQuestionChannelTrace) (string, *vo.SubQuestionChannelTrace) {
		return t.ChannelName, t
	})

	for _, rawResult := range rawResults {
		channelName := rawResult.ChannelName
		execution := &vo.ChannelExecutionView{
			ExchangeId:       trace.ExchangeId(),
			TraceId:          trace.TraceId(),
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
	trace.RecordChannelExecutions(executions)
	return nil
}

// recordRetrievalResultObservations 记录检索结果观测数据，包括各阶段分数、是否通过闸门、是否被选中等
func (e *RetrievalImpl) recordRetrievalResultObservations(trace *vo.ConversationTrace, subQuestionIndex int, subQuestion string,
	rawResults, filteredResults []*vo.RetrievalChannelResult, finalDocuments []*vo.DocumentChunk) error {
	if len(rawResults) == 0 {
		return nil
	}

	// 构建最终文档ID到排名的映射
	finalRankMap := make(map[string]int)
	for i, doc := range finalDocuments {
		finalRankMap[doc.ID] = i + 1
	}

	// 构建通过闸门的文档ID集合
	gatePassedSet := make(map[string]map[string]bool)
	for _, fr := range filteredResults {
		for _, doc := range fr.Documents {
			gatePassedSet[fr.ChannelName][doc.ID] = true
		}
	}

	results := make([]*vo.RetrievalResultView, 0)
	for _, rawResult := range rawResults {
		channelName := rawResult.ChannelName
		for i, doc := range rawResult.Documents {
			view := &vo.RetrievalResultView{
				ExchangeId:       trace.ExchangeId(),
				TraceId:          trace.TraceId(),
				SubQuestionIndex: subQuestionIndex,
				SubQuestion:      subQuestion,
				ChannelType:      channelName,
				ChannelRank:      i + 1,
				GatePassed:       gatePassedSet[channelName][doc.ID],
			}

			view.SetDocumentInfo(doc)

			// 判断是否被选中
			if rank, ok := finalRankMap[doc.ID]; ok {
				view.Selected = true
				view.FinalRank = rank
				view.SelectionReason = "已选入最终 Prompt"
			} else if !view.GatePassed {
				// 未通过闸门
				if vo.RetrievalChannelVector == channelName {
					view.SelectionReason = fmt.Sprintf("向量闸门过滤：分数 %.4f < 阈值 %.4f",
						view.OriginalScore, e.minVectorSimilarity)
				} else if vo.RetrievalChannelKeyword == channelName {
					view.SelectionReason = fmt.Sprintf("关键词闸门过滤：分数 %.4f 低于相对阈值（floor=%.2f）",
						view.OriginalScore, e.keywordRelativeScoreFloor)
				} else {
					view.SelectionReason = fmt.Sprintf("闸门过滤：分数 %.4f", view.OriginalScore)
				}
			} else {
				// 超出finalTopK限制
				view.SelectionReason = fmt.Sprintf("超出 finalTopK 限制（topK=%d）", e.finalTopK)
			}
			results = append(results, view)
		}
	}

	// todo 待完善最终保存
	trace.RecordRetrievalResults(results)
	return nil
}

// renderParentEvidenceText 渲染父级证据文本：[父块内容] + [命中子片段]
func (e *RetrievalImpl) renderParentEvidenceText(parentBlock *entity.DocumentParentBlock, childDocuments []*vo.DocumentChunk, maxChars int) string {
	parentText := strutil.Trim(parentBlock.ParentText)

	// 当父块无内容时，使用首条子文档的内容作为回退
	if strutil.IsBlank(parentText) {
		return utils.Ternary(len(childDocuments) > 0, childDocuments[0].OriginalSnippet, "")
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

func Warnf(format string, args ...interface{}) {
	logx.Alert(fmt.Sprintf(format, args...))
}
