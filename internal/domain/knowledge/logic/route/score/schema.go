package score

// Features 包含所有参与评分的原始维度，各字段取值范围需外部归一化。
type Features struct {
	SemanticScore      float64 // 语义相似度，通常 0~1
	LexicalScore       float64 // 词索引（如 BM25）得分，可为任意正值
	RelationScore      float64 // 持久化关系得分，如 topic-document 关系分
	ScopeRelationScore float64 // 可选，如 topic 与 scope 的匹配关系（0/1 或加权）
}

func NewFeatures(semanticScore, lexicalScore, relationScore, scopeRelationScore float64) *Features {
	return &Features{
		SemanticScore:      max(0, semanticScore),
		LexicalScore:       max(0, lexicalScore),
		RelationScore:      max(0, relationScore),
		ScopeRelationScore: max(0, scopeRelationScore),
	}
}

// Result 评分结果
type Result struct {
	TotalScore float64            // 最终总分
	Reason     string             // 可读的召回原因
	Source     string             // 主要来源：SEMANTIC / ROUTE_INDEX / PERSISTED_RELATION / COMPOSITE / NONE
	Features   map[string]float64 // 各维度原始分及中间值，便于调试
}
