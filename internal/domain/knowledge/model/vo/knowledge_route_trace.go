package vo

// KnowledgeRouteTrace 知识路由跟踪记录
type KnowledgeRouteTrace struct {
	ID                  int64
	ConversationId      string
	ExchangeId          string
	Question            string
	RewriteQuestion     string
	Mode                string // auto / shadow
	TopScopesJson       string
	TopTopicsJson       string
	TopDocumentsJson    string
	SelectedDocumentId  int64
	HitSelectedDocument *int
	Confidence          float64
	RouteStatus         string // SUCCESS / LOW_CONFIDENCE / FAILED
	ErrorMsg            string
	Status              int
}
