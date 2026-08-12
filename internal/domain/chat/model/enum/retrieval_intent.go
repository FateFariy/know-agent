package enum

type RetrievalIntent = string

const (
	RetrievalIntentGeneral   RetrievalIntent = "GENERAL"
	RetrievalIntentTable     RetrievalIntent = "TABLE"
	RetrievalIntentGraphRAG  RetrievalIntent = "GRAPH_RAG"
	RetrievalIntentRaptor    RetrievalIntent = "RAPTOR"
	RetrievalIntentStructure RetrievalIntent = "STRUCTURE"
)
