package intent

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// NavigationIndexer 章节索引器（与结构图谱并列定位章节）
type NavigationIndexer interface {
	// SearchSections 按关键词+维度检索匹配的章节命中
	SearchSections(ctx context.Context, input *SearchInput) ([]*NavigationSectionHit, error)
}

// Recognizer 意图识别策略接口
type Recognizer interface {
	// Name 返回提供者名称，用于日志和溯源
	Name() string

	// Recognize 根据输入的意图识别输入，返回意图识别结果
	Recognize(ctx context.Context, input *RecognitionInput) (*vo.IntentRecognitionResult, error)
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
