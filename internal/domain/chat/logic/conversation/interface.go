package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/aggregate"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// Stage 表示对话流程中的一个阶段
type Stage interface {
	// Name 返回阶段名称
	Name() string

	// Execute 执行阶段逻辑
	// ctx: 标准上下文，用于控制取消、超时和传递请求作用域的值
	// convCtx: 对话上下文，携带会话状态和业务数据
	// sink: 事件输出器，用于发送流式事件
	Execute(ctx context.Context, convCtx *Context) error
}

// ConditionalStage 条件执行阶段
type ConditionalStage interface {
	Stage

	// ShouldExecute 决定是否执行该阶段
	ShouldExecute(ctx context.Context, convCtx *Context) bool
}

// SessionMemoryManager 会话记忆管理器
type SessionMemoryManager interface {
	// LoadMemoryContext 加载会话记忆上下文
	LoadMemoryContext(ctx context.Context, conversationId string) (*aggregate.Conversation, error)

	// RefreshConversationSummaryAsync 异步刷新会话摘要
	RefreshConversationSummaryAsync(conversationId string)

	// GetConversationSummary 获取会话摘要
	GetConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// RebuildConversationSummary 重建会话摘要
	RebuildConversationSummary(ctx context.Context, conversationId string) (*entity.ChatMemorySummary, error)

	// DeleteConversationSummary 删除会话摘要
	DeleteConversationSummary(ctx context.Context, conversationId string) error
}

// QueryRewriter 查询改写器
type QueryRewriter interface {
	Rewrite(ctx context.Context, question, historySummary string) (*vo.QuestionRewriteResult, error)
}

type RecognitionInput struct {
	OriginalQuestion         string
	RewrittenQuestion        string
	SubQuestions             []string
	HistorySummary           string
	RecentQuestionTranscript string
}

// Recognizer 意图识别策略接口
type Recognizer interface {
	// Name 返回提供者名称，用于日志和溯源
	Name() string

	// Recognize 根据输入的意图识别输入，返回意图识别结果
	Recognize(ctx context.Context, input *RecognitionInput) (*vo.IntentRecognitionResult, error)
}

type KnowledgeRouteInput struct {
	ConversationId             string
	ExchangeId                 int64
	Question                   string
	RewriteQuestion            string
	KnowledgeBaseSelectionMode string
	SelectedDocumentId         int64
	SelectedKnowledgeBaseIds   []int64
	SelectedKnowledgeBaseNames []string
	AllowedDocumentIds         []int64
}

// KnowledgeRouter 知识路由器
type KnowledgeRouter interface {
	// Route 根据问题进行知识路由
	Route(ctx context.Context, input *KnowledgeRouteInput) (*vo.KnowledgeRouteDecision, error)

	// RecordShadowRoute 记录影子路由结果
	RecordShadowRoute(ctx context.Context, input *KnowledgeRouteInput) error
}

func NewKnowledgeRouteInput(convCtx *Context, rewriteQuestion string) *KnowledgeRouteInput {
	mapper := func(doc *vo.DocumentMetadata) int64 { return doc.DocumentId }
	return &KnowledgeRouteInput{
		ConversationId:             convCtx.ConversationId,
		ExchangeId:                 convCtx.ExchangeId,
		Question:                   convCtx.Question,
		RewriteQuestion:            rewriteQuestion,
		KnowledgeBaseSelectionMode: convCtx.KnowledgeBaseSelectionSnapshot.SelectionModeName(),
		SelectedDocumentId:         convCtx.SelectedDocumentId,
		SelectedKnowledgeBaseIds:   convCtx.KnowledgeBaseSelectionSnapshot.SelectedKnowledgeBaseIds,
		SelectedKnowledgeBaseNames: convCtx.KnowledgeBaseSelectionSnapshot.SelectedKnowledgeBaseNames,
		AllowedDocumentIds:         utils.Map(convCtx.KnowledgeBaseSelectionSnapshot.AllowedDocuments, mapper),
	}
}

// DocumentRouteInput 文档路由输入
type DocumentRouteInput struct {
	DocumentId        int64                       `json:"documentId"`              // 文档ID
	OriginalQuestion  string                      `json:"originalQuestion"`        // 原始问题
	RewriteResult     *vo.QuestionRewriteResult   `json:"rewriteResult"`           // 改写结果
	RecognitionResult *vo.IntentRecognitionResult `json:"intentRecognitionResult"` // 查询理解结果
}

// RewrittenQuestion 获取改写后问题，无改写则回退原始问题
func (d *DocumentRouteInput) RewrittenQuestion() string {
	if d == nil {
		return ""
	}
	question := d.OriginalQuestion
	if d.RewriteResult != nil && utils.IsNotBlank(d.RewriteResult.RewrittenQuestion) {
		question = d.RewriteResult.RewrittenQuestion
	}
	return utils.Trim(question)
}

// SubQuestions 获取子问题列表
func (d *DocumentRouteInput) SubQuestions() []string {
	if d == nil || d.RewriteResult == nil {
		return nil
	}
	return d.RewriteResult.NormalizeSubQuestions(d.RewrittenQuestion())
}

// RouteText 获取路由文本（原始 + 改写）
func (d *DocumentRouteInput) RouteText() string {
	if d == nil {
		return ""
	}
	return utils.Trim(d.OriginalQuestion) + " " + d.RewrittenQuestion()
}

// DocumentRouter 文档路由器
type DocumentRouter interface {
	// Route 根据文档ID和问题进行文档内路由
	Route(ctx context.Context, input *DocumentRouteInput) (*vo.DocumentNavigationDecision, error)
}

// Retriever RAG 检索引擎接口
type Retriever interface {
	Retrieve(ctx context.Context, plan *vo.RetrievalPlan) (*vo.RetrievalResult, error)
}

// QuestionRecommender 追问推荐器
type QuestionRecommender interface {
	// Generate 生成推荐追问
	Generate(ctx context.Context, question, answer string, recentExchanges []*entity.ChatExchange) ([]string, error)
}

// CacheHit 语义缓存命中结果
type CacheHit struct {
	Entry      *entity.ChatCacheEntry
	Similarity float32
}

// SearchInput 聚合 ANN 检索所需的查询参数
type SearchInput struct {
	QueryText string         // 查询文本
	Threshold float32        // 相似度阈值
	Scope     *vo.CacheScope // 检索作用域
}

// SemanticCacheStore 语义缓存存储接口
type SemanticCacheStore interface {
	// Search 在 scope 内对查询文本做 ANN 检索，返回相似度达标的最相似完整
	Search(ctx context.Context, input *SearchInput) (*CacheHit, error)

	// Put 写入/更新一条缓存
	Put(ctx context.Context, entry *entity.ChatCacheEntry) error
}
