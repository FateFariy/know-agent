package vo

type RetrievalChannel = string

const (
	RetrievalChannelVector  = "indexer" // 向量检索
	RetrievalChannelKeyword = "keyword" // 关键词检索
	RetrievalChannelRerank  = "rerank"  // 重排序
	RetrievalChannelHybrid  = "hybrid"  // 混合检索
)

func RetrievalChannelCode(channel RetrievalChannel) int {
	switch channel {
	case RetrievalChannelVector:
		return 1
	case RetrievalChannelKeyword:
		return 2
	case RetrievalChannelRerank:
		return 3
	case RetrievalChannelHybrid:
		return 4
	default:
		return 0
	}
}
