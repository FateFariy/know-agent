package graph

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// AnswerRender 图谱回答渲染接口
type AnswerRender interface {
	RenderAnswer(mode enum.ExecutionMode, decision *vo.DocumentNavigationDecision, result *entity.GraphQueryResult) string
}

// GraphQuerier 结构图查询接口
type GraphQuerier interface {
	// ListSections 列出指定文档下的所有结构图节点（用于本地短语匹配）
	ListSections(ctx context.Context, documentId int64) ([]*entity.GraphSection, error)

	FindSectionById(ctx context.Context, documentId int64, nodeId int64) (*entity.GraphSection, error)

	// FindSectionByCode 根据编号（如 1.2.3 / 第 3 节）匹配章节节点
	FindSectionByCode(ctx context.Context, documentId int64, sectionCode string) (*entity.GraphSection, error)

	// FindBestSection 根据问题文本查找最佳节点；可接受一个可选的 anchor 短语增强
	FindBestSection(ctx context.Context, documentId int64, question, anchorHint string) (*entity.GraphSection, error)

	// FindSectionWithChildren 根据节点编号查找子节点
	FindSectionWithChildren(ctx context.Context, documentId int64, sectionNodeId int64) (*entity.GraphSectionWithChildren, error)

	// FindSectionWithSiblings 根据节点编号查找同级节点
	FindSectionWithSiblings(ctx context.Context, documentId int64, sectionNodeId int64) (*entity.GraphSectionWithSiblings, error)

	// BuildGraphResult 根据节点编号构建结构图结果
	BuildGraphResult(ctx context.Context, documentId int64, sectionNodeId int64, itemIndex int, itemKeyword string) (*entity.GraphQueryResult, error)
}
