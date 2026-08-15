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

//func RetrievalChannelCode(channel RetrievalChannel) int {
//	switch channel {
//	case RetrievalChannelVector:
//		return 1
//	case RetrievalChannelKeyword:
//		return 2
//	case RetrievalChannelRerank:
//		return 3
//	case RetrievalChannelHybrid:
//		return 4
//	default:
//		return 0
//	}
//}
