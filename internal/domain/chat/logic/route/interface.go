package route

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// NavigationSectionHit 章节索引服务返回的命中节点
type NavigationSectionHit struct {
	NodeId int64   // 结构图谱中的节点 ID
	Score  float64 // 命中分数
}

// SearchInput 搜索输入参数
type SearchInput struct {
	DocumentId      int64  // 文档 ID
	Topic           string // 主题/关键词
	Facet           string // 维度："章节" / "步骤" / ""
	InformationNeed string // 信息需求描述（用于语义匹配）
	Question        string // 用户原始问题
	TopK            int    // 返回结果数量上限
}

// NavigationIndexer 章节索引器（与结构图谱并列定位章节）
type NavigationIndexer interface {
	// SearchSections 按关键词+维度检索匹配的章节命中
	SearchSections(ctx context.Context, input *SearchInput) ([]*NavigationSectionHit, error)
}

// DocumentRouter 文档路由器
type DocumentRouter interface {
	// Route 根据文档ID和问题进行文档内路由
	Route(ctx context.Context, input *DocumentRouteInput) (*vo.DocumentNavigationDecision, error)
}
