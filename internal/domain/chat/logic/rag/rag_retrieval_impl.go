package rag

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/stream"
	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	doclog "github.com/swiftbit/know-agent/internal/domain/document/logic"
	den "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	dvo "github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

const rrfK = 60

type RetrievalImpl struct {
	repo                      adapter.ChatRepository
	reranker                  adapter.Reranker
	channels                  []RetrievalChannel
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

func NewRetrievalImpl(svcCtx *svc.ServiceContext, repo adapter.ChatRepository, reranker adapter.Reranker,
	channels []RetrievalChannel, documentLogic doclog.LifecycleLogic) *RetrievalImpl {
	return &RetrievalImpl{
		repo:                      repo,
		channels:                  channels,
		reranker:                  reranker,
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

var _ logic.RagRetrieveLogic = (*RetrievalImpl)(nil)

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
			start := time.Now()
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

			rerankedCandidates := e.applyRerank(ctx, ragCtx, parentSearchDocs, subQuestion)

			finalTopK := min(e.finalTopK, len(rerankedCandidates))
			finalDocuments := rerankedCandidates[:finalTopK]

			ragCtx.AddRetrievalNotef("子问题%d检索完成：%s，final=%d",
				subQuestionIndex, e.summarizeChannelResults(filteredResults), len(finalDocuments))

			// 记录观测数据
			if trace != nil {
				if err = e.recordChannelObservations(ctx, trace, subQuestionIndex, subQuestion, start, rawChannelResults, filteredResults, channelTraces); err != nil {
					Warnf("记录通道观测数据失败: subQuestionIndex=%d, error=%v", subQuestionIndex, err)
				}
				if err = e.recordRetrievalResultObservations(ctx, trace, subQuestionIndex, subQuestion, rawChannelResults, filteredResults, finalDocuments); err != nil {
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
	channels := slice.Filter(e.channels, func(_ int, item RetrievalChannel) bool { return item.Supports(plan) })
	if len(channels) == 0 {
		return nil, nil
	}

	// 创建带缓冲的结果通道，容量 = 通道数量，避免 goroutine 写阻塞
	resultCh := make(chan *vo.RetrievalChannelResult, len(channels))
	defer close(resultCh)

	// 为每个通道启动一个 goroutine 并行执行检索
	for _, ch := range channels {
		go func(ch RetrievalChannel) {
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
func (e *RetrievalImpl) retrieveChannel(ctx context.Context, ch RetrievalChannel, query *vo.DocumentRetrieve) (*vo.RetrievalChannelResult, error) {
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
	parentBlockMap := utils.SliceToMapBy(parentBlocks, func(item *den.DocumentParentBlock) (int64, *den.DocumentParentBlock) {
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
func (e *RetrievalImpl) buildParentEvidenceDocument(parentBlock *den.DocumentParentBlock, childDocuments []*vo.DocumentChunk, maxChars int) *vo.DocumentChunk {
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
		IsElevated:        1,
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

// applyRerank 应用重排序
func (e *RetrievalImpl) applyRerank(ctx context.Context, ragCtx *vo.RagRetrievalContext, candidates []*vo.DocumentChunk, subQuestion string) []*vo.DocumentChunk {
	if !e.rerankEnabled || len(candidates) == 0 || e.reranker == nil {
		return candidates
	}

	ragCtx.AddUsedChannel(vo.RetrievalChannelRerank)
	result, err := e.reranker.Process(ctx, subQuestion, candidates)
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

// recordChannelObservations 记录渠道执行观测数据：将原始/过滤结果与追踪信息汇总为 ChatChannelExecution 记录
//
// 为每个渠道产出一条观测，字段涵盖：
//   - 会话/交互/追踪 ID、子问题索引与内容
//   - 执行起止时间、DurationMs、ExecutionState 状态码
//   - RecalledCount（原始召回数量）、AcceptedCount（过滤后数量）、FinalSelectedCount（最终选中数量）
//   - 渠道文档分数（SetScores），从 rawResult 计算得出
//
// 执行流程：
//  1. 空结果快速返回
//  2. 将过滤结果与渠道追踪记录分别转为 ChannelName → 对象的 map，加速后续 join
//  3. 遍历原始结果：为每个渠道构建 ChatChannelExecution，补充过滤后/最终选中数量
//  4. 调用 repo 批量写入数据库
func (e *RetrievalImpl) recordChannelObservations(ctx context.Context, trace *vo.ConversationTrace, subQuestionIndex int, subQuestion string,
	start time.Time, rawResults, filteredResults []*vo.RetrievalChannelResult, channelTraces []*vo.SubQuestionChannelTrace) error {
	if len(rawResults) == 0 {
		return nil
	}

	// 结束时间 + 预分配执行列表
	end := time.Now()
	executions := make([]*vo.ChatChannelExecution, 0, len(rawResults))

	// 将过滤结果 / 渠道追踪记录转为 map，供按 channelName 快速定位
	filteredResultsMap := utils.SliceToMapBy(filteredResults, func(r *vo.RetrievalChannelResult) (string, *vo.RetrievalChannelResult) {
		return r.ChannelName, r
	})
	channelTracesMap := utils.SliceToMapBy(channelTraces, func(t *vo.SubQuestionChannelTrace) (string, *vo.SubQuestionChannelTrace) {
		return t.ChannelName, t
	})

	// 遍历每个渠道的原始结果，构建一条观测记录
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

		// 从过滤结果补 AcceptedCount（过滤后保留的文档数量）
		if filteredResult, ok := filteredResultsMap[channelName]; ok {
			execution.AcceptedCount = len(filteredResult.Documents)
		}
		// 从渠道追踪记录补 FinalSelectedCount（最终选入 Prompt 的数量）
		if trace, ok := channelTracesMap[channelName]; ok {
			execution.FinalSelectedCount = trace.AcceptedCount
		}

		// 设置渠道文档分数（由 rawResult.Documents 计算）
		execution.SetScores(rawResult.Documents)
		executions = append(executions, execution)
	}

	// 批量写入数据库
	return e.repo.InsertChannelExecutions(ctx, executions)
}

// recordRetrievalResultObservations 记录检索结果观测数据：对每个渠道的每个原始文档生成一条 ChatRetrievalResult，
// 用于追踪在"原始召回 → 过滤/闸门 → 最终选择"全流程中每篇文档的状态与原因。
//
// 核心字段：
//   - ChannelRank：该渠道内原始排名（从 1 起）
//   - RrfRank：RRF 融合后的排名（按 RRFScore 降序）
//   - GatePassed：是否通过对应渠道的过滤/闸门（1=通过，0=未通过）
//   - IsSelected：是否被选入最终 Prompt；FinalRank：在最终 Prompt 中的排名
//   - SelectionReason：未被选中的原因（闸门过滤/超出 topK/其他），用于离线分析
//
// 执行流程：
//  1. 空结果快速返回
//  2. 构建最终文档 ID → FinalRank 的映射（保留原始传入顺序，从 1 起编号）
//  3. 按 RRFScore 降序排序 finalDocuments，再构建 RrfRank 映射（RRF 融合后的排名）
//  4. 构建"通过闸门"的文档 ID 集合（以渠道名分组）
//  5. 遍历所有渠道的所有原始文档：
//     - 填充会话/子问题/渠道/排名等基础信息
//     - 调用 SetDocumentInfo 写入文档信息（ID/标题/分数等）
//     - 判定是否被最终选中；未选中则按渠道类型格式化原因
//  6. 调用 repo 批量写入数据库
func (e *RetrievalImpl) recordRetrievalResultObservations(ctx context.Context, trace *vo.ConversationTrace, subQuestionIndex int, subQuestion string,
	rawResults, filteredResults []*vo.RetrievalChannelResult, finalDocuments []*vo.DocumentChunk) error {
	if len(rawResults) == 0 {
		return nil
	}

	// 基于传入的 finalDocuments 顺序构建 FinalRank 映射（保留调用方的选择顺序，从 1 起编号）
	finalRankMap := make(map[string]int)
	for i, doc := range finalDocuments {
		finalRankMap[doc.ID] = i + 1
	}

	// 按 RRFScore 降序排序 finalDocuments，再构建 RrfRank 映射（RRF 融合后的排名）
	sort.Slice(finalDocuments, func(i, j int) bool {
		return finalDocuments[i].RRFScore > finalDocuments[j].RRFScore
	})
	rrkRankMap := make(map[string]int)
	for i, doc := range finalDocuments {
		rrkRankMap[doc.ID] = i + 1
	}

	// 构建"通过闸门"的文档 ID 集合（按渠道名分组）—— filteredResults 即通过闸门的结果集
	gatePassedSet := make(map[string]map[string]int)
	for _, fr := range filteredResults {
		gatePassedSet[fr.ChannelName] = make(map[string]int)
		for _, doc := range fr.Documents {
			// 按渠道名建立文档 ID → 1 的映射（存在即表示通过）
			gatePassedSet[fr.ChannelName][doc.ID] = 1
		}
	}

	// 遍历每个渠道的每个原始文档，生成 ChatRetrievalResult 观测记录
	results := make([]*vo.ChatRetrievalResult, 0)
	for _, rawResult := range rawResults {
		channelName := rawResult.ChannelName
		for i, doc := range rawResult.Documents {
			// 构建基础信息（会话、子问题、渠道、渠道内排名、RRF 排名、闸门通过状态）
			view := &vo.ChatRetrievalResult{
				ConversationId:   trace.ConversationId(),
				ExchangeId:       trace.ExchangeId(),
				TraceId:          trace.TraceId(),
				SubQuestionIndex: subQuestionIndex,
				SubQuestion:      subQuestion,
				ChannelType:      channelName,
				ChannelRank:      i + 1,
				RrfRank:          rrkRankMap[doc.ID],
				GatePassed:       gatePassedSet[channelName][doc.ID],
			}

			// 填充文档元信息（ID / 标题 / 原文摘要 / 原始分数等）
			view.SetDocumentInfo(doc)

			// 判定是否被选入最终 Prompt；否则写入原因
			if rank, ok := finalRankMap[doc.ID]; ok {
				// 已选入最终 Prompt：设置 IsSelected=1 与 FinalRank
				view.IsSelected = 1
				view.FinalRank = rank
				view.SelectionReason = "已选入最终 Prompt"
			} else if view.GatePassed == 0 {
				// 未通过闸门：按渠道类型格式化原因
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
				// 通过了闸门但因超出 finalTopK 限制而未被选中
				view.SelectionReason = fmt.Sprintf("超出 finalTopK 限制（topK=%d）", e.finalTopK)
			}
			results = append(results, view)
		}
	}

	// 批量写入数据库
	return e.repo.InsertRetrievalResults(ctx, results)
}

// renderParentEvidenceText 渲染父级证据文本：[父块内容] + [命中子片段]
func (e *RetrievalImpl) renderParentEvidenceText(parentBlock *den.DocumentParentBlock, childDocuments []*vo.DocumentChunk, maxChars int) string {
	parentText := strutil.Trim(parentBlock.ParentText)

	// 当父块无内容时，使用首条子文档的内容作为回退
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

func Warnf(format string, args ...interface{}) {
	logx.Alert(fmt.Sprintf(format, args...))
}
