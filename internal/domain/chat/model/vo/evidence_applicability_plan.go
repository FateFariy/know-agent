package vo

import "strings"

// EvidenceApplicabilityPlan 证据适用性计划
type EvidenceApplicabilityPlan struct {
	TargetEntities   []string `json:"targetEntities"`   // 目标实体列表
	ExcludedEntities []string `json:"excludedEntities"` // 排除实体列表
	SelectionReason  string   `json:"selectionReason"`  // 选择原因
	Source           string   `json:"source"`           // 来源
}

// NewEvidenceApplicabilityPlan 构建证据适用性计划
func NewEvidenceApplicabilityPlan(question string, intentResult *IntentRecognitionResult) *EvidenceApplicabilityPlan {
	plan := &EvidenceApplicabilityPlan{
		SelectionReason: "",
		Source:          "",
	}

	if intentResult != nil {
		// 计算导航锚点状态
		plan.AnchoredNav = intentResult.HasAnchor()
		plan.StrongNavIntent = intentResult.IsNavigational()

		// 根据置信度决定是否需要文档级证据
		confidence := intentResult.GetConfidence()
		normalizedConfidence := normalizeConfidence(confidence)
		plan.NeedsDocumentLevel = normalizedConfidence >= 0.72

		plan.SelectionReason = strings.TrimSpace(strings.Join(intentResult.Channels, ", "))
	}

	// 构建候选通道列表
	plan.ChannelCandidates = append(plan.ChannelCandidates,
		RetrievalChannelPlan{Name: "VECTOR", Enabled: true, Weight: 0.8},
		RetrievalChannelPlan{Name: "KEYWORD", Enabled: true, Weight: 0.7},
	)

	if !plan.NeedsDocumentLevel {
		plan.SelectionReason = "Navigational intent not detected; evidence will be gathered across all channels."
	} else {
		plan.SelectionReason = "Document-level evidence was selected based on navigational intent confidence."
	}

	return plan
}

// normalizeConfidence 归一化置信度
func normalizeConfidence(confidence float64) float64 {
	if confidence > 1 {
		return confidence / 100
	}
	if confidence < 0 {
		return 0
	}
	return confidence
}
