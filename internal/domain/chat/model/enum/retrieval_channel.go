package enum

// RetrievalChannel 检索通道枚举
type RetrievalChannel = string

const (
	RetrievalChannelVector   RetrievalChannel = "vector"    // 向量检索
	RetrievalChannelKeyword  RetrievalChannel = "keyword"   // 关键词检索
	RetrievalChannelTable    RetrievalChannel = "table"     // 表格检索
	RetrievalChannelGraphRAG RetrievalChannel = "graph_rag" // 图谱RAG检索
	RetrievalChannelRaptor   RetrievalChannel = "raptor"    // RAPTOR检索
)

// intentToChannelsMap 定义意图到渠道的映射关系
var intentToChannelsMap = map[RetrievalIntent][]RetrievalChannel{
	RetrievalIntentGeneral:  {RetrievalChannelVector, RetrievalChannelKeyword},
	RetrievalIntentTable:    {RetrievalChannelTable},
	RetrievalIntentGraphRAG: {RetrievalChannelGraphRAG},
	RetrievalIntentRaptor:   {RetrievalChannelRaptor},
}

// ConvertIntentsToChannels 将检索意图列表转换为检索渠道列表（去重）
func ConvertIntentsToChannels(intents []RetrievalIntent) []RetrievalChannel {
	if len(intents) == 0 {
		return nil
	}

	seen := make(map[RetrievalChannel]bool)
	result := make([]RetrievalChannel, 0)

	for _, intent := range intents {
		for _, ch := range intentToChannelsMap[intent] {
			if !seen[ch] {
				seen[ch] = true
				result = append(result, ch)
			}
		}
	}
	return result
}
