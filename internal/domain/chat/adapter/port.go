package adapter

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// CheckPointStore 检查点存储器
type CheckPointStore interface {
	// Get 获取检查点
	Get(ctx context.Context, checkPointId string) ([]byte, bool, error)

	// Set 设置检查点
	Set(ctx context.Context, checkPointId string, checkPoint []byte) error

	// Count 检查点数量
	Count(ctx context.Context, checkPointId string) (int, error)

	// Delete 删除检查点
	Delete(ctx context.Context, checkPointId string) (int, error)
}

type DistributedLock interface {
	// TryLock 尝试获取锁
	TryLock(ctx context.Context, name string) error

	// Lock 获取锁
	Lock(ctx context.Context, name string) error

	// UnlockContext 释放锁
	UnlockContext(ctx context.Context, name string) error

	// Unlock 释放锁
	Unlock(name string) error

	// Extend 锁续期
	Extend(ctx context.Context, name string) error
}

type KeywordRetriever interface {
	// SearchByKeyword  按关键词检索
	SearchByKeyword(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}

type VectorRetriever interface {
	// SearchByVector 按向量检索
	SearchByVector(ctx context.Context, query *vo.DocumentRetrieve) ([]*vo.DocumentChunk, error)
}

type DocumentFetcher interface {
	// FetchRetrieveDocuments 获取文档
	FetchRetrieveDocuments(ctx context.Context) ([]*vo.DocumentMetadata, error)

	// QueryParentChunks 查询父块
	QueryParentChunks(ctx context.Context, id string) ([]*vo.DocumentChunk, error)
}

// Sink 面向客户端的事件输出端口
type Sink interface {
	// Text 发送文本事件
	Text(content string, conversationId string, exchangeId int64) error

	// Thinking 发送思考中事件
	Thinking(content string, conversationId string, exchangeId int64) error

	// Status 发送状态事件
	Status(content string, conversationId string, exchangeId int64) error

	// Error 发送错误事件
	Error(content string, conversationId string, exchangeId int64) error

	// References 发送参考事件
	References(references []*vo.SearchReference, conversationId string, exchangeId int64) error

	// Recommendations 发送推荐事件
	Recommendations(recommendations []string, conversationId string, exchangeId int64) error

	// Finish 发送完成事件
	Finish(conversationId string, exchangeId int64) error
}

type Renderer interface {
	Render(templateName string, variables map[string]any) (string, error)
}

// KnowledgeRouter 知识路由器
type KnowledgeRouter interface {
	// Route 根据问题进行知识路由
	Route(ctx context.Context, question, rewriteQuestion string) (*vo.KnowledgeRouteDecision, error)

	// RecordAutoRoute 记录自动路由结果
	RecordAutoRoute(ctx context.Context, exchangeId int64, conversationId, question, rewriteQuestion string, decision *vo.KnowledgeRouteDecision) error

	// RecordShadowRoute 记录影子路由结果
	RecordShadowRoute(ctx context.Context, exchangeId, documentId int64, conversationId, question, rewriteQuestion string) error
}
