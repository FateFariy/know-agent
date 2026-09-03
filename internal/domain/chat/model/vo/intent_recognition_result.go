package vo

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
)

// IntentRecognitionResult 意图识别结果
type IntentRecognitionResult struct {
	QueryType                 enum.QueryType                `json:"queryType,omitempty"`
	Channels                  []enum.RetrievalIntent        `json:"channels,omitempty"`
	Entities                  []string                      `json:"entities,omitempty"`
	TargetEntities            []string                      `json:"targetEntities,omitempty"`
	ExcludedEntities          []string                      `json:"excludedEntities,omitempty"`
	SectionAnchors            []string                      `json:"sectionAnchors,omitempty"`
	StructureNavigationIntent *StructureNavigationIntent    `json:"structureNavigationIntent,omitempty"`
	TableOps                  []string                      `json:"tableOps,omitempty"`
	AnswerShapePlan           []enum.AnswerShapeRequirement `json:"answerShapePlan,omitempty"`
	Confidence                float64                       `json:"confidence,omitempty"`
	Reasons                   []string                      `json:"reasons,omitempty"`
	Source                    string                        `json:"source,omitempty"`
}

// IsFollowUpQuestion 判断是否为追问
func (i *IntentRecognitionResult) IsFollowUpQuestion(question string) bool {
	if utils.IsBlank(question) {
		return false
	}
	return i != nil && i.QueryType == enum.QueryTypeFollowUp
}

// IsStructureNavigationConfident 判断结构导航意图是否置信度高
func (i *IntentRecognitionResult) IsStructureNavigationConfident(threshold float64) bool {
	if i != nil && i.QueryType == enum.QueryTypeStructureNavigation &&
		i.StructureNavigationIntent.IsConfident(threshold) {
		return true
	}
	return false
}

// ResolveAction 解析结构导航动作
func (i *IntentRecognitionResult) ResolveAction(threshold float64) string {
	if i == nil || !i.IsStructureNavigationConfident(threshold) {
		return ""
	}
	return i.StructureNavigationIntent.ResolveAction(threshold)
}

// HasSectionAnchor 判断是否包含有效的章节锚点
func (i *IntentRecognitionResult) HasSectionAnchor() bool {
	if i == nil || len(i.SectionAnchors) == 0 {
		return false
	}
	for _, anchor := range i.SectionAnchors {
		if utils.IsNotBlank(anchor) {
			return true
		}
	}
	return false
}

// GetEntities 获取实体列表
func (i *IntentRecognitionResult) GetEntities() []string {
	if i == nil {
		return nil
	}
	return i.Entities
}

// HasAnchor 判断是否包含结构导航锚点
func (i *IntentRecognitionResult) HasAnchor() bool {
	if i == nil || i.StructureNavigationIntent == nil {
		return false
	}
	return utils.IsNotBlank(i.StructureNavigationIntent.AnchorSectionPath) ||
		utils.IsNotBlank(i.StructureNavigationIntent.AnchorCanonicalPath) ||
		i.StructureNavigationIntent.AnchorStructureNodeId > 0
}

// IsNavigational 判断是否为导航意图
func (i *IntentRecognitionResult) IsNavigational() bool {
	if i == nil {
		return false
	}
	return i.QueryType == enum.QueryTypeStructureNavigation
}

// GetStructureNavigationIntent 获取结构导航意图
func (i *IntentRecognitionResult) GetStructureNavigationIntent() *StructureNavigationIntent {
	if i == nil {
		return nil
	}
	return i.StructureNavigationIntent
}

func (i *IntentRecognitionResult) SuggestedChannels() []string {
	if i == nil || len(i.Channels) == 0 {
		return nil
	}
	return utils.FilterUniqueLimit(i.Channels, -1, func(s string) (string, bool) { return s, s != "" })
}

func (i *IntentRecognitionResult) ToTableIntent() *TableIntent {
	if i == nil {
		return &TableIntent{
			Source: "NONE",
		}
	}
	return &TableIntent{
		Requested: i.channelRequested(enum.QueryTypeTableQuery, "TABLE"),
		TableOps:  normalizeStringsLimit(i.TableOps, 8),
		Source:    i.intentSource(),
	}
}

func (i *IntentRecognitionResult) ToGraphIntent(maxHops int) *GraphIntent {
	if i == nil {
		return &GraphIntent{
			MaxHops: maxHops,
			Source:  "NONE",
		}
	}
	return &GraphIntent{
		Requested:      i.channelRequested("GRAPH_RELATION", "GRAPH_RAG"),
		Entities:       normalizeStringsLimit(i.Entities, 8),
		TargetEntities: normalizeStringsLimit(i.TargetEntities, 8),
		MaxHops:        maxHops,
		Source:         i.intentSource(),
	}
}

func (i *IntentRecognitionResult) ToRaptorIntent(sourceChunkTopK int) *RaptorIntent {
	if i == nil {
		return &RaptorIntent{
			SourceChunkTopK: sourceChunkTopK,
			Source:          "NONE",
		}
	}
	requested := i.channelRequested("GLOBAL_SUMMARY", "RAPTOR")
	return &RaptorIntent{
		Requested:        requested,
		SummaryRequested: requested,
		SourceChunkTopK:  sourceChunkTopK,
		Source:           i.intentSource(),
	}
}

// channelRequested 检查通道是否被请求
func (i *IntentRecognitionResult) channelRequested(queryType, intent string) bool {
	if i == nil {
		return false
	}
	return i.QueryType == queryType || utils.ContainsAny(i.Channels, intent)
}

// intentSource 获取意图来源
func (i *IntentRecognitionResult) intentSource() string {
	if i == nil {
		return "NONE"
	}
	return utils.BlankToDefault(i.Source, "intent-recognize")
}

// normalizeStringsLimit 标准化字符串列表并限制数量
func normalizeStringsLimit(items []string, limit int) []string {
	return utils.FilterMapUniqueLimit(items, limit, func(item string) (string, string, bool) {
		item = utils.CompactWhitespace(item)
		return item, item, item != ""
	})
}
