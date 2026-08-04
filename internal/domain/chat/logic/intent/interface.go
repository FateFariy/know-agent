package intent

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// DocumentRouter 文档路由器
type DocumentRouter interface {
	// Route 根据文档ID和问题进行文档内路由
	Route(ctx context.Context, documentId int64, question string, rewriteResult *vo.QuestionRewriteResult) (*vo.DocumentNavigationDecision, error)
}

// NavigationIndexService 可选的章节索引服务接口（与结构图谱并列定位章节）
type NavigationIndexService interface {
	// SearchSections 按关键词+维度检索匹配的章节命中
	SearchSections(ctx context.Context, documentId int64, topic, facet, informationNeed, question string, topK int) ([]*NavigationSectionHit, error)
}
