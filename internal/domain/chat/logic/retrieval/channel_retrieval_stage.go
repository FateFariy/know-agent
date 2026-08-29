package retrieval

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/adapter"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChannelRetrievalStage 并行调用各检索通道，收集原始检索结果
type ChannelRetrievalStage struct {
	channels   []Retrieval
	docGateway adapter.DocumentGateway
}

// NewChannelRetrievalStage 创建通道检索阶段
func NewChannelRetrievalStage(channels []Retrieval, docGateway adapter.DocumentGateway) *ChannelRetrievalStage {
	return &ChannelRetrievalStage{
		channels:   channels,
		docGateway: docGateway,
	}
}

func (s *ChannelRetrievalStage) Name() string {
	return "ChannelRetrieval"
}

// Execute 并行检索所有支持的通道，结果写入 state.ChannelResults，并提前回填原始文档元数据
func (s *ChannelRetrievalStage) Execute(ctx context.Context, state *RetrievalState) error {
	retrievalResult := state.RetrievalResult
	subQuestionIndex := state.Input.SubQuestionIndex
	subQuestion := state.Input.SubQuestion

	// 过滤出当前计划需要的通道
	channels := utils.Filter(s.channels, func(item Retrieval) bool { return utils.ContainsAny(state.Input.EnableChannels(), item.Name()) })
	if len(channels) == 0 {
		retrievalResult.AddRetrievalNotef("子问题%d没有可用的检索通道。", subQuestionIndex)
		return nil
	}
	channelPlanMap := state.Input.ChannelPlanMap()

	// 创建带缓冲的结果通道
	resultCh := make(chan *RetrievalChannelResult, len(channels))
	defer close(resultCh)

	// 并行检索单个子问题的所有通道
	for _, ch := range channels {
		go func(ch Retrieval) {
			timeoutCtx, cancel := context.WithTimeout(ctx, channelPlanMap[ch.Name()].Timeout)
			defer cancel()
			result, err := ch.Retrieve(timeoutCtx, state.Input)
			if err != nil {
				logx.Warnf("检索通道失败: subQuestionIndex=%d, subQuestion='%s', channel='%s', error=%v",
					subQuestionIndex, subQuestion, ch.Name(), err)
				retrievalResult.AddRetrievalNotef("子问题%d通道[%s]检索失败或超时，已自动降级。", subQuestionIndex, ch.Name())
				result = &RetrievalChannelResult{Name: ch.Name()}
			}
			resultCh <- result
		}(ch)
	}

	// 主循环收集结果
	channelResults := make([]*RetrievalChannelResult, 0, len(channels))
	channelTraces := make([]*vo.SubQuestionChannelTrace, 0, len(channels))
	for result := range resultCh {
		channelResults = append(channelResults, result)
		channelTraces = append(channelTraces, &vo.SubQuestionChannelTrace{
			ChannelName:     result.Name,
			RecalledCount:   len(result.RawDocuments),
			AcceptedCount:   len(result.AcceptedDocuments),
			RetrievalIntent: state.Plan.PrimaryIntent,
			ChannelWeight:   channelPlanMap[result.Name].Weight,
		})
		if len(result.AcceptedDocuments) > 0 {
			state.RetrievalResult.AddUsedChannel(result.Name)
		}
		if len(channelResults) == len(channels) {
			break
		}
	}
	state.ChannelResults = channelResults
	state.ChannelTraces = channelTraces

	// 检索完成后提前回填各通道召回的原始文档元数据
	s.enrichRawDocumentsMetadata(ctx, state)

	return nil
}

// enrichRawDocumentsMetadata 检索完成后为各通道召回的原始文档补全知识库元数据
func (s *ChannelRetrievalStage) enrichRawDocumentsMetadata(ctx context.Context, state *RetrievalState) {
	seen := make(map[int64]struct{}, 64)
	documentIds := make([]int64, 0, 64)
	for _, result := range state.ChannelResults {
		for _, doc := range result.RawDocuments {
			if _, ok := seen[doc.DocumentId]; !ok && doc.NeedsMetadataFallback() {
				seen[doc.DocumentId] = struct{}{}
				documentIds = append(documentIds, doc.DocumentId)
			}
		}
	}
	if len(documentIds) == 0 {
		return
	}

	metadata, err := s.docGateway.FindRetrieveDocumentByIds(ctx, documentIds...)
	if err != nil {
		logx.Warnf("获取知识库元数据失败: documentIds=%v, error=%v", documentIds, err)
		return
	}
	metadataMap := utils.MapBy(metadata, func(item *vo.DocumentMetadata) (int64, *vo.DocumentMetadata) {
		return item.DocumentId, item
	})
	for _, result := range state.ChannelResults {
		for _, doc := range result.RawDocuments {
			doc.EnrichFromMetadata(metadataMap[doc.DocumentId])
		}
	}
}
