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
	return queryType == "FOLLOW_UP"
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
