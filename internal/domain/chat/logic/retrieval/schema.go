package retrieval

import (
	"fmt"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// RetrievalChannelResult 检索通道结果
type RetrievalChannelResult struct {
	Name              string              `json:"name"`
	RawDocuments      []*vo.DocumentChunk `json:"rawDocuments"`
	AcceptedDocuments []*vo.DocumentChunk `json:"acceptedDocuments"`
}

// ExecutionInput 不可变检索执行请求
type ExecutionInput struct {
	SubQuestionIndex int
	SubQuestion      string
	ContextHints     []string
	ScopeMode        string
	KnowledgeBaseIds []int64
	DocumentScope    []int64
	TaskScope        []int64
	Filters          *vo.RetrievalMetadataFilters
	Channels         []*vo.RetrievalChannelPlan
	TableIntent      *vo.TableIntent  // 表格意图（暂未使用）
	GraphIntent      *vo.GraphIntent  // 图意图（暂未使用）
	RaptorIntent     *vo.RaptorIntent // Raptor意图（暂未使用）
}

// newRetrievalExecutionInput 从 RetrievalPlan 和 RetrievalExecutionQuery 编译检索执行请求
func newRetrievalExecutionInput(plan *vo.RetrievalPlan, query *vo.RetrievalExecutionQuery) (*ExecutionInput, error) {
	if plan == nil {
		return nil, fmt.Errorf("RetrievalPlan is required")
	}
	if query == nil {
		return nil, fmt.Errorf("RetrievalExecutionQuery is required")
	}
	plannedQuery, err := plan.RequirePlannedQuery(query.Index)
	if err != nil {
		return nil, err
	}
	if !plannedQuery.Equal(query) {
		return nil, fmt.Errorf("execution query does not match RetrievalPlan question plan")
	}
	channels := enum.ConvertIntentsToChannels(plan.SuggestedIntents)
	if len(channels) == 0 {
		channels = append(channels, enum.RetrievalChannelKeyword, enum.RetrievalChannelVector)
	}
	channelPlans := make([]*vo.RetrievalChannelPlan, 0, len(channels))
	for _, ch := range plan.Channels {
		if utils.ContainsAny(channels, ch.Name) {
			channelPlans = append(channelPlans, ch.Clone())
		}
	}

	input := &ExecutionInput{
		SubQuestionIndex: query.Index,
		SubQuestion:      query.SubQuestion,
		ContextHints:     utils.Copy(query.ContextHints),
		ScopeMode:        plan.ScopeMode,
		KnowledgeBaseIds: utils.Copy(plan.KnowledgeBaseIds),
		DocumentScope:    utils.Copy(plan.DocumentScope),
		TaskScope:        utils.Copy(plan.TaskScope),
		Filters:          plan.MetadataFilters.Clone(),
		Channels:         channelPlans,
		TableIntent:      plan.TableIntent.Clone(),
		GraphIntent:      plan.GraphIntent.Clone(),
		RaptorIntent:     plan.RaptorIntent.Clone(),
	}

	if err = input.validate(); err != nil {
		return nil, err
	}
	return input, nil
}

// validate 校验检索执行请求字段
func (r *ExecutionInput) validate() error {
	if r.SubQuestionIndex <= 0 || utils.IsBlank(r.SubQuestion) {
		return fmt.Errorf("execution input query index and text are required")
	}
	if r.ScopeMode == "" || r.ScopeMode == enum.KbSelectionModeNone {
		return fmt.Errorf("execution input scope mode is required")
	}
	if len(r.DocumentScope) == 0 || len(r.DocumentScope) != len(r.TaskScope) {
		return fmt.Errorf("execution input document and task scope must be non-empty and aligned")
	}
	if len(r.Channels) == 0 {
		return fmt.Errorf("execution input channels must be non-empty and unique")
	}
	names := make(map[string]struct{}, len(r.Channels))
	for _, ch := range r.Channels {
		if ch == nil {
			return fmt.Errorf("execution input channel must not be nil")
		}
		if _, exists := names[ch.Name]; exists {
			return fmt.Errorf("execution input channel names must be unique")
		}
		names[ch.Name] = struct{}{}
	}
	return nil
}

// RequireChannel 按名称查找通道，不存在则返回错误
func (r *ExecutionInput) RequireChannel(channelName string) (*vo.RetrievalChannelPlan, error) {
	for _, ch := range r.Channels {
		if ch.Name == channelName {
			return ch, nil
		}
	}
	return nil, fmt.Errorf("retrieval channel is absent from execution input: %s", channelName)
}

func (r *ExecutionInput) EnableChannels() []string {
	if r == nil {
		return nil
	}
	return utils.Map(r.Channels, func(ch *vo.RetrievalChannelPlan) string { return ch.Name })
}

// ChannelPlanMap 返回通道计划的映射，键为通道名称
func (r *ExecutionInput) ChannelPlanMap() map[string]*vo.RetrievalChannelPlan {
	if r == nil || len(r.Channels) == 0 {
		return nil
	}
	keyFunc := func(ch *vo.RetrievalChannelPlan) (string, *vo.RetrievalChannelPlan) { return ch.Name, ch }
	return utils.MapBy(r.Channels, keyFunc)
}
