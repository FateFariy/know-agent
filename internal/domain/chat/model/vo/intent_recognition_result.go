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
	queryType := ""
	if i != nil {
		queryType = i.QueryType
	}
	return queryType == enum.QueryTypeFollowUp
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

// PrimaryRetrievalIntent 获取主要检索意图
func (i *IntentRecognitionResult) PrimaryRetrievalIntent() enum.RetrievalIntent {
	if i == nil {
		return enum.RetrievalIntentGeneral
	}
	queryType := utils.BlankToDefault(i.QueryType, enum.QueryTypeDocumentQA)
	switch queryType {
	case enum.QueryTypeStructureNavigation:
		return enum.RetrievalIntentStructure
	case enum.QueryTypeTableQuery:
		return enum.RetrievalIntentTable
	case enum.QueryTypeGraphRelation:
		return enum.RetrievalIntentGraphRAG
	case enum.QueryTypeGlobalSummary:
		return enum.RetrievalIntentRaptor
	}
	// 从通道列表获取
	for _, ch := range i.Channels {
		switch ch {
		case enum.RetrievalIntentTable:
			return enum.RetrievalIntentTable
		case enum.RetrievalIntentGraphRAG:
			return enum.RetrievalIntentGraphRAG
		case enum.RetrievalIntentRaptor:
			return enum.RetrievalIntentRaptor
		case enum.RetrievalIntentStructure:
			return enum.RetrievalIntentStructure
		}
	}
	return enum.RetrievalIntentGeneral
}

// GetConfidence 获取置信度
func (i *IntentRecognitionResult) GetConfidence() float64 {
	if i == nil {
		return 0
	}
	return i.Confidence
}

// NormalizeConfidence 归一化置信度到[0, 1]范围
func (i *IntentRecognitionResult) NormalizeConfidence() float64 {
	if i == nil {
		return 0
	}
	confidence := i.Confidence
	if confidence > 1 {
		return confidence / 100.0
	}
	if confidence < 0 {
		return 0
	}
	return confidence
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

// ResolveNoEvidenceReply 根据是否需要新鲜搜索返回合适的无证据回复
func (i *IntentRecognitionResult) ResolveNoEvidenceReply(requiresFreshSearch bool) string {
	queryType := utils.BlankToDefault(i.QueryType, enum.QueryTypeDocumentQA)
	if queryType == enum.QueryTypeCapabilityQuery {
		return "当前你正在使用“当前文档问答”模式，我会优先基于所选文档回答。这个问题更像是在询问助手能力，而不是当前文档内容。如果你想了解我能做什么，请切换到“开放式提问”模式。"
	}
	if queryType == enum.QueryTypeOpenChat || requiresFreshSearch {
		return "当前你正在使用“当前文档问答”模式，我只能基于所选文档回答。这个问题更像开放式提问，例如天气、最新信息或一般交流。如果你想继续问这类问题，请切换到“开放式提问”模式。"
	}
	return "当前没有从当前文档中检索到足够证据，暂时不能给出可靠结论。你可以补充更具体的标题、术语或关键词后再试。"
}
