package enum

import (
	"strings"

	"github.com/swiftbit/know-agent/common/utils"
)

type RetrievalIntent = string

const (
	RetrievalIntentGeneral   RetrievalIntent = "general"
	RetrievalIntentTable     RetrievalIntent = "table"
	RetrievalIntentGraphRAG  RetrievalIntent = "graph_rag"
	RetrievalIntentRaptor    RetrievalIntent = "raptor"
	RetrievalIntentStructure RetrievalIntent = "structure"
)

// ParseRetrievalIntents 解析字符串列表为 RetrievalIntent 列表，去重并忽略无效值
// 特殊处理：vector/bm25/keyword 映射为 general
func ParseRetrievalIntents(raws []string) []RetrievalIntent {
	if len(raws) == 0 {
		return nil
	}
	seen := make(map[RetrievalIntent]bool)
	var result []RetrievalIntent

	for _, raw := range raws {
		normalized := strings.ToUpper(utils.Trim(raw))
		if normalized == "" {
			continue
		}

		// vector/bm25/keyword 映射为 general
		if normalized == "vector" || normalized == "bm25" || normalized == "keyword" {
			if !seen[RetrievalIntentGeneral] {
				seen[RetrievalIntentGeneral] = true
				result = append(result, RetrievalIntentGeneral)
			}
			continue
		}

		// 检查是否属于已知类型
		switch normalized {
		case RetrievalIntentGeneral, RetrievalIntentTable,
			RetrievalIntentGraphRAG, RetrievalIntentRaptor,
			RetrievalIntentStructure:
			if !seen[normalized] {
				seen[normalized] = true
				result = append(result, normalized)
			}
		}
	}
	return result
}
