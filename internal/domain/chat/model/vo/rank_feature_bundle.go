package vo

// RankFeatureBundle 排序特征包
type RankFeatureBundle struct {
	EnabledFeatures     []string `json:"enabledFeatures"`     // 启用的特征列表
	RankWeight          float64  `json:"rankWeight"`          // 排序权重
	OriginalScoreWeight float64  `json:"originalScoreWeight"` // 原始分数权重
	MetadataBoostWeight float64  `json:"metadataBoostWeight"` // 元数据提升权重
	MaxMetadataBoost    float64  `json:"maxMetadataBoost"`    // 最大元数据提升
	Source              string   `json:"source"`              // 来源
}

// BuildRankFeatures 构建排序特征包
func BuildRankFeatures(hybrid *HybridOptions) *RankFeatureBundle {
	if hybrid == nil {
		hybrid = &HybridOptions{}
	}
	return &RankFeatureBundle{
		EnabledFeatures:     []string{"CHANNEL_RRF", "ORIGINAL_SCORE", "PERSISTED_METADATA"},
		RankWeight:          hybrid.RankWeight,
		OriginalScoreWeight: hybrid.OriginalScoreWeight,
		MetadataBoostWeight: hybrid.MetadataBoostWeight,
		MaxMetadataBoost:    hybrid.MaxMetadataBoost,
		Source:              "PERSISTED_INDEX_METADATA",
	}
}
