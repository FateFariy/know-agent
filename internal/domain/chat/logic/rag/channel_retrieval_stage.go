package rag

import (
	"context"

	"github.com/swiftbit/know-agent/common/logx"
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChannelRetrievalStage 并行调用各检索通道，收集原始检索结果
type ChannelRetrievalStage struct {
	channels []Retrieval
}

// NewChannelRetrievalStage 创建通道检索阶段
func NewChannelRetrievalStage(channels []Retrieval) *ChannelRetrievalStage {
	return &ChannelRetrievalStage{
		channels: channels,
	}
}

func (s *ChannelRetrievalStage) Name() string {
	return "ChannelRetrieval"
}

// Execute 并行检索所有支持的通道，结果写入 state.ChannelResults
func (s *ChannelRetrievalStage) Execute(ctx context.Context, state *RetrievalState) error {
	retrievalResult := state.RetrievalResult
	subQuestionIndex := state.Input.SubQuestionIndex
	subQuestion := state.Input.ExecutionQuery

	// 过滤出当前计划需要的通道
	channels := utils.Filter(s.channels, func(item Retrieval) bool { return utils.ContainsAny(state.Input.EnableChannels(), item.Name()) })
	if len(channels) == 0 {
		retrievalResult.AddRetrievalNotef("子问题%d没有可用的检索通道。", subQuestionIndex)
		return nil
	}
	channelPlanMap := utils.MapBy(state.Input.Channels, func(item *vo.RetrievalChannelPlan) (string, *vo.RetrievalChannelPlan) {
		return item.Channel, item
	})

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
	for result := range resultCh {
		channelResults = append(channelResults, result)
		if len(channelResults) == len(channels) {
			break
		}
	}
	state.ChannelResults = channelResults
	state.ChannelTraces = s.buildChannelTraces(channelResults, state.Plan)

	return nil
}

// buildChannelTraces 构建子问题渠道执行追踪
func (s *ChannelRetrievalStage) buildChannelTraces(results []*RetrievalChannelResult, plan *vo.RetrievalPlan) []*vo.SubQuestionChannelTrace {
	if len(results) == 0 {
		return nil
	}

	rawMap := make(map[string]int)
	acceptedMap := make(map[string]int)
	channelNames := make(map[string]struct{})

	utils.ForEach(results, func(index int, r *RetrievalChannelResult) {
		rawMap[r.Name] = len(r.RawDocuments)
		acceptedMap[r.Name] = len(r.AcceptedDocuments)
		channelNames[r.Name] = struct{}{}
	})

	return utils.Map(utils.MapKeys(channelNames), func(name string) *vo.SubQuestionChannelTrace {
		return &vo.SubQuestionChannelTrace{
			ChannelName:     name,
			RetrievalIntent: plan.PrimaryIntent,
			RecalledCount:   rawMap[name],
			AcceptedCount:   acceptedMap[name],
		}
	})
}
