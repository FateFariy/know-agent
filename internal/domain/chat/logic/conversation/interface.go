package conversation

import (
	"context"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/route"
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

type RouteInput struct {
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
	Route(ctx context.Context, input *RouteInput) (*vo.KnowledgeRouteDecision, error)

	// RecordShadowRoute 记录影子路由结果
	RecordShadowRoute(ctx context.Context, input *RouteInput) error
}

func NewRouteInput(convCtx *Context, rewriteQuestion string) *RouteInput {
	mapper := func(doc *vo.DocumentMetadata) int64 { return doc.DocumentId }
	return &RouteInput{
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

// DocumentRouter 文档路由器
type DocumentRouter interface {
	// Route 根据文档ID和问题进行文档内路由
	Route(ctx context.Context, input *route.DocumentRouteInput) (*vo.DocumentNavigationDecision, error)
}
