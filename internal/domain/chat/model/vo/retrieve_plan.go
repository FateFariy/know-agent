package vo

import (
	"errors"
	"fmt"
	"time"
)

type RetrievalPlan struct {
	QuestionPlan              *RetrievalQuestionPlan       // 检索问题计划
	ChatMode                  string                       // 对话查询模式
	PrimaryIntent             string                       // 主要检索意图
	SuggestedIntents          []string                     // 建议的检索意图列表
	ScopeMode                 string                       // 知识库选择模式
	KnowledgeBaseIds          []int64                      // 知识库ID列表
	DocumentScope             []int64                      // 文档范围ID列表
	TaskScope                 []int64                      // 任务范围ID列表
	MetadataFilters           *RetrievalMetadataFilters    // 元数据过滤条件
	Channels                  []*RetrievalChannelPlan      // 检索通道计划列表
	NavigationAction          string                       // 文档导航动作
	StructureNavigationResult *StructureNavigationResult   // 结构导航结果
	StructureAnchor           *ConversationStructureAnchor // 会话结构锚点
	ItemAnchor                *ConversationItemAnchor      // 会话条目锚点
	TableIntent               *TableIntent                 // 表格检索意图
	GraphIntent               *GraphIntent                 // 图谱检索意图
	RaptorIntent              *RaptorIntent                // RAPTOR检索意图
	RankFeatures              *RankFeatureBundle           // 排序特征包
	CandidateTopK             int                          // 候选窗口大小
	RerankTopK                int                          // 重排序窗口大小
	RerankEnabled             bool                         // 是否请求重排序
	FinalTopK                 int                          // 最终证据预算
	SubQuestionTimeout        time.Duration                // 子问题超时时间
}

func (p *RetrievalPlan) Validate() error {
	if p == nil {
		return errors.New("retrieval plan is nil")
	}
	return nil
}

// FindPlannedQuery 按索引查找执行查询
func (p *RetrievalPlan) FindPlannedQuery(queryIndex int) *RetrievalExecutionQuery {
	if p == nil || p.QuestionPlan == nil || len(p.QuestionPlan.ExecutionQueries) == 0 {
		return nil
	}
	for _, q := range p.QuestionPlan.ExecutionQueries {
		if q != nil && q.Index == queryIndex {
			return q
		}
	}
	return nil
}

// RequirePlannedQuery 按索引查找执行查询，不存在则返回错误
func (p *RetrievalPlan) RequirePlannedQuery(queryIndex int) (*RetrievalExecutionQuery, error) {
	query := p.FindPlannedQuery(queryIndex)
	if query == nil {
		return nil, fmt.Errorf("execution query index is absent from RetrievalPlan: %d", queryIndex)
	}
	return query, nil
}
