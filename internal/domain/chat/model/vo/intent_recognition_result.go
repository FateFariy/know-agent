package vo

import (
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
