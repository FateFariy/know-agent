package intent

import (
	"context"
)

// DefaultNavigationIndexService 默认的章节索引服务（空实现，保留未来扩展）。
// DocumentQuestionRouter 中 NavigationIndexService 为可选依赖，缺失时会跳过索引检索路径。
type DefaultNavigationIndexService struct {
}

// NewDefaultNavigationIndexService 创建默认的章节索引服务实现。
func NewDefaultNavigationIndexService() *DefaultNavigationIndexService {
	return &DefaultNavigationIndexService{}
}

var _ NavigationIndexService = (*DefaultNavigationIndexService)(nil)

// SearchSections 空实现——预留接口，未来可接入向量/关键词索引系统。
func (s *DefaultNavigationIndexService) SearchSections(ctx context.Context, documentId int64, topic, facet, informationNeed, question string, topK int) ([]*NavigationSectionHit, error) {
	return nil, nil
}
