package retrieval

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/duke-git/lancet/v2/slice"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter/rerank"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

// Engine RAG 检索引擎实现
type Engine struct {
	pipeline *Pipeline
}

func NewRetrievalEngine(svcCtx *svc.ServiceContext, repo adapter.ChatRepository, reranker rerank.Reranker,
	channels []Retrieval, docGateway adapter.DocumentGateway, fusion Fusion) *Engine {
	maxChars := svcCtx.Config.Chat.Rag.ParentEvidenceMaxChars
	pipeline := NewPipeline(
		NewChannelRetrievalStage(channels, docGateway),
		NewFusionStage(fusion),
		NewParentElevationStage(docGateway, maxChars),
		NewRerankStage(reranker),
		NewFinalTopKStage(),
		NewTopDocumentSummaryStage(),
		NewObservationStage(repo),
	)
	return &Engine{
		pipeline: pipeline,
	}
}

// Retrieve RAG 检索入口，执行多子问题并行检索
func (e *Engine) Retrieve(ctx context.Context, plan *vo.RetrievalPlan) (*vo.RetrievalResult, error) {
	if err := plan.Validate(); err != nil {
		return nil, err
	}

	retrievalResult := vo.NewRagRetrievalResult(plan.QuestionPlan.Question)

	inputs := make([]*ExecutionInput, 0, len(plan.QuestionPlan.ExecutionQueries))
	for _, query := range plan.QuestionPlan.ExecutionQueries {
		input, err := newRetrievalExecutionInput(plan, query)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}

	evidenceList := e.retrieveSubQuestionParallel(ctx, retrievalResult, inputs, plan)
	acceptedCount := slice.CountBy(evidenceList, func(index int, item *vo.SubQuestionEvidence) bool { return len(item.SourceDocuments) > 0 })

	logx.Infof("RAG 检索完成: retrievalQuestion='%s', originalSubQuestionCount=%d, acceptedSubQuestionCount=%d, notes=%v",
		retrievalResult.RetrievalQuestion, len(evidenceList), acceptedCount, retrievalResult.RetrievalNotes())

	e.assignReferenceIds(evidenceList)
	retrievalResult.SubQuestionEvidenceList = evidenceList

	return retrievalResult, nil
}

// -------------------- 子问题并行检索 --------------------

// retrieveSubQuestionParallel 并行检索所有子问题，每个子问题通过管线独立执行完整检索流程
func (e *Engine) retrieveSubQuestionParallel(ctx context.Context, retrievalResult *vo.RetrievalResult, inputs []*ExecutionInput, plan *vo.RetrievalPlan) []*vo.SubQuestionEvidence {
	timeoutCtx, cancel := context.WithTimeout(ctx, plan.SubQuestionTimeout)
	defer cancel()

	resultChan := make(chan *vo.SubQuestionEvidence, len(inputs))
	defer close(resultChan)

	for _, input := range inputs {
		go func(input *ExecutionInput) {
			subQuestionIndex := input.SubQuestionIndex
			subQuestion := input.SubQuestion
			start := time.Now()
			state := &RetrievalState{
				Input:           input,
				RetrievalResult: retrievalResult,
				Plan:            plan,
				Start:           start,
			}

			if err := e.pipeline.Execute(timeoutCtx, state); err != nil {
				logx.Warnf("子问题检索失败: subQuestionIndex=%d, subQuestion='%v", subQuestionIndex, err)
				retrievalResult.AddRetrievalNotef("子问题%d检索失败或超时，已自动忽略。", subQuestionIndex)
				resultChan <- &vo.SubQuestionEvidence{SubQuestionIndex: subQuestionIndex, SubQuestion: subQuestion}
				return
			}

			resultChan <- &vo.SubQuestionEvidence{
				SubQuestionIndex:       subQuestionIndex,
				SubQuestion:            subQuestion,
				SourceDocuments:        state.FinalDocs,
				ChannelTraces:          state.ChannelTraces,
				FusedCandidateCount:    len(state.FusedDocs),
				ParentCandidateCount:   len(state.ParentSearchDocs),
				RerankedCandidateCount: len(state.RerankedDocs),
			}
		}(input)
	}

	evidenceList := make([]*vo.SubQuestionEvidence, 0, len(inputs))
	for result := range resultChan {
		evidenceList = append(evidenceList, result)
		if len(evidenceList) == len(inputs) {
			break
		}
	}
	slices.SortFunc(evidenceList, func(a, b *vo.SubQuestionEvidence) int {
		return a.SubQuestionIndex - b.SubQuestionIndex
	})

	return evidenceList
}

// -------------------- 引用 ID 分配 --------------------

// assignReferenceIds 为检索证据分配引用 ID
func (e *Engine) assignReferenceIds(evidenceList []*vo.SubQuestionEvidence) {
	referenceNumber := 1
	assignedIds := make(map[string]string)

	for _, evidence := range evidenceList {
		references := make([]*vo.SearchReference, 0, len(evidence.SourceDocuments))
		for _, doc := range evidence.SourceDocuments {
			ref := doc.ToSearchReference(evidence.SubQuestionIndex, 0, evidence.SubQuestion)
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
