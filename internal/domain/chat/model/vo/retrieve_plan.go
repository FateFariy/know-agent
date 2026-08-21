package vo

import (
	"errors"
	"fmt"
	"math"
	"time"
)

type RetrievalPlan struct {
	QuestionPlan              *RetrievalQuestionPlan       // 检索问题计划
	ChatMode                  string                       // 对话查询模式
	PrimaryIntent             string                       // 主要检索意图
	SuggestedIntents          []string                     // 建议的检索意图列表
	ScopeMode                 string                       // 知识库选择模式
	KnowledgeBaseIds          []int64                      // 知识库ID列表
	AllowedDocumentScope      []int64                      // 允许的文档范围ID列表
	DocumentScope             []int64                      // 文档范围ID列表
	TaskScope                 []int64                      // 任务范围ID列表
	MetadataFilters           *RetrievalMetadataFilters    // 元数据过滤条件
	EvidenceApplicabilityPlan *EvidenceApplicabilityPlan   // 证据适用性计划
	Channels                  []*RetrievalChannelPlan      // 检索通道计划列表
	StructureNavigation       *StructureNavigationIntent   // 结构导航意图
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

func (p *RetrievalPlan) hasRetrievalQuestion() bool {
	return p.QuestionPlan != nil && p.QuestionPlan.RetrievalQuestion != ""
}

// RankWeight 返回排名权重（默认 1）
func (p *RetrievalPlan) RankWeight() float64 {
	if p == nil || p.RankFeatures == nil {
		return 1
	}
	return math.Max(0, p.RankFeatures.RankWeight)
}

// OriginalScoreWeight 返回原始分数权重（默认 0.08）
func (p *RetrievalPlan) OriginalScoreWeight() float64 {
	if p == nil || p.RankFeatures == nil {
		return 0.08
	}
	return math.Max(0, p.RankFeatures.OriginalScoreWeight)
}

// MetadataBoostWeight 返回元数据提升权重（默认 0.04）
func (p *RetrievalPlan) MetadataBoostWeight() float64 {
	if p == nil || p.RankFeatures == nil {
		return 0.04
	}
	return math.Max(0, p.RankFeatures.MetadataBoostWeight)
}

// MaxMetadataBoost 返回最大元数据提升值（默认 1）
func (p *RetrievalPlan) MaxMetadataBoost() float64 {
	if p == nil || p.RankFeatures == nil {
		return 1
	}
	return math.Max(0, p.RankFeatures.MaxMetadataBoost)
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

func (p *RetrievalPlan) validateRouteAuthorization() error {
	//mode := p.RoutePlan.AuthorizationMode
	//authDocs := p.RoutePlan.AuthorizedDocumentIds
	//authTasks := p.RoutePlan.AuthorizedTaskIds
	//
	//if mode == "" || isBlank(p.RoutePlan.ScopeAuthorizationReason) || authDocs == nil || authTasks == nil {
	//	return errors.New("RetrievalPlan route authorization is required")
	//}
	//if !int64SlicesEqual(p.DocumentScope, authDocs) || !int64SlicesEqual(p.TaskScope, authTasks) {
	//	return errors.New("RetrievalPlan route authorization scope must equal execution scope")
	//}
	//
	//topDoc := p.RoutePlan.TopDocumentHintId
	//topTask := p.RoutePlan.TopTaskHintId
	//if (topDoc == nil) != (topTask == nil) {
	//	return errors.New("RetrievalPlan route recommendation document/task hint must be paired")
	//}
	//if topDoc != nil {
	//	idx := indexOfInt64(p.DocumentScope, *topDoc)
	//	if idx < 0 || idx >= len(p.TaskScope) || p.TaskScope[idx] != *topTask {
	//		return errors.New("RetrievalPlan route recommendation hint must match an authorized document/task pair")
	//	}
	//}
	//
	//switch mode {
	//case "EXPLICIT_DOCUMENT":
	//	if p.ChatMode != "DOCUMENT" || len(p.DocumentScope) != 1 ||
	//		topDoc == nil || *topDoc != p.DocumentScope[0] {
	//		return errors.New("explicit document authorization requires one matching DOCUMENT scope")
	//	}
	//case "KNOWLEDGE_BASE_ALLOWED_SCOPE":
	//	if p.ChatMode != "AUTO_DOCUMENT" || !int64SlicesEqual(p.DocumentScope, p.AllowedDocumentScope) {
	//		return errors.New("knowledge-base authorization requires the complete AUTO_DOCUMENT allowed scope")
	//	}
	//case "CLARIFICATION_REQUIRED":
	//	return errors.New("clarification-required route authorization is not executable")
	//}
	return nil
}
