package vo

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/swiftbit/know-agent/common/utils"
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
	RoutePlan                 *RetrievalRoutePlan          // 路由计划
	RankFeatures              *RankFeatureBundle           // 排序特征包
	CandidateWindow           int                          // 候选窗口大小
	RerankWindow              int                          // 重排序窗口大小
	RerankRequested           bool                         // 是否请求重排序
	FinalEvidenceBudget       int                          // 最终证据预算
	SubQuestionTimeout        time.Duration                // 子问题超时时间
	Reasons                   []string                     // 决策原因列表
	Source                    string                       // 来源标识
}

func (p *RetrievalPlan) Validate() error {
	if p == nil {
		return errors.New("retrieval plan is nil")
	}
	return p.ValidateForExecution()
}

//// CandidateWindow 返回候选窗口大小（非负）
//func (p *RetrievalPlan) CandidateWindow() int {
//	if p == nil || p.CandidateWindow <= 0 {
//		return 0
//	}
//	return p.CandidateWindow
//}

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

// ValidateForExecution 校验执行计划的完整性和合法性
func (p *RetrievalPlan) ValidateForExecution() error {
	//if p == nil {
	//	return errors.New("RetrievalPlan is required")
	//}
	//if p.QuestionPlan == nil || utils.IsBlank(p.QuestionPlan.RetrievalQuestion) {
	//	return errors.New("RetrievalPlan normalized query is required")
	//}
	//
	//// 校验执行查询列表
	//if err := p.validateExecutionQueries(); err != nil {
	//	return err
	//}
	//
	//// 校验各类 ID 列表必须为正数
	//if err := requirePositiveIDs(p.KnowledgeBaseIds, "knowledge base scope"); err != nil {
	//	return err
	//}
	//if err := requirePositiveIDs(p.AllowedDocumentScope, "allowed document scope"); err != nil {
	//	return err
	//}
	//if err := requirePositiveIDs(p.DocumentScope, "document scope"); err != nil {
	//	return err
	//}
	//if err := requirePositiveIDs(p.TaskScope, "task scope"); err != nil {
	//	return err
	//}
	//
	//// 校验作用域模式
	//if p.ScopeMode == enum.KbSelectionModeNone {
	//	return errors.New("RetrievalPlan knowledge base scope mode is required")
	//}
	//
	//// 校验文档作用域包含关系
	//if !utils.ContainsAll(p.AllowedDocumentScope, p.DocumentScope) {
	//	return errors.New("RetrievalPlan document scope must stay inside allowed document scope")
	//}
	//
	//// 校验文档与任务作用域数量一致
	//if len(p.DocumentScope) != len(p.TaskScope) {
	//	return errors.New("RetrievalPlan document scope and task scope must have the same size")
	//}
	//
	//// 校验核心策略对象非空
	//if p.ChatMode == "" || p.PrimaryIntent == "" || p.MetadataFilters == nil ||
	//	p.EvidenceApplicabilityPlan == nil || p.RoutePlan == nil || p.RankFeatures == nil ||
	//	p.TableIntent == nil || p.GraphIntent == nil || p.RaptorIntent == nil {
	//	return errors.New("RetrievalPlan controlled filters, applicability, route, intents and rank features are required")
	//}
	//
	//// 校验路由授权（假设该方法已存在，不自行实现）
	//if err := p.validateRouteAuthorization(); err != nil {
	//	return err
	//}
	//
	//// 校验通道计划
	//if err := p.validateChannels(); err != nil {
	//	return err
	//}
	//
	//// 校验窗口和预算参数
	//if p.CandidateWindow <= 0 {
	//	return errors.New("RetrievalPlan candidate window must be positive")
	//}
	//if p.RerankWindow <= 0 || p.RerankWindow > p.CandidateWindow {
	//	return errors.New("RetrievalPlan rerank window must be positive and no larger than candidate window")
	//}
	//if p.FinalEvidenceBudget <= 0 || p.FinalEvidenceBudget > p.RerankWindow {
	//	return errors.New("RetrievalPlan final evidence budget must be positive and no larger than rerank window")
	//}
	//if p.SubQuestionTimeout <= 0 {
	//	return errors.New("RetrievalPlan sub-question timeout must be positive")
	//}

	return nil
}

// validateExecutionQueries 校验执行查询列表
func (p *RetrievalPlan) validateExecutionQueries() error {
	if len(p.QuestionPlan.ExecutionQueries) == 0 {
		return errors.New("RetrievalPlan execution query is required for every sub-question")
	}
	for _, q := range p.QuestionPlan.ExecutionQueries {
		if q == nil || utils.IsBlank(q.ExecutionQuery) {
			return errors.New("RetrievalPlan execution query is required for every sub-question")
		}
	}
	return nil
}

// validateChannels 校验通道配置
func (p *RetrievalPlan) validateChannels() error {
	//if len(p.Channels) == 0 {
	//	return errors.New("RetrievalPlan channel plans are required")
	//}
	//
	//channelNames := make(map[string]struct{}, len(p.Channels))
	//for _, ch := range p.Channels {
	//	if ch == nil {
	//		return errors.New("RetrievalPlan channel names must be non-blank and unique")
	//	}
	//	if strings.TrimSpace(ch.Name) == "" {
	//		return errors.New("RetrievalPlan channel names must be non-blank and unique")
	//	}
	//	if _, exists := channelNames[ch.Name]; exists {
	//		return errors.New("RetrievalPlan channel names must be non-blank and unique")
	//	}
	//	channelNames[ch.Name] = struct{}{}
	//
	//	if ch.TopK <= 0 || ch.Budget <= 0 || ch.Timeout <= 0 {
	//		return errors.New("RetrievalPlan channel topK, budget and timeout must be positive")
	//	}
	//	if !isFiniteAndNonNegative(ch.Weight) {
	//		return errors.New("RetrievalPlan channel weight must be finite and non-negative")
	//	}
	//	if !isFiniteAndNonNegative(ch.MinimumScore) || !isFiniteAndNonNegative(ch.RelativeScoreFloor) {
	//		return errors.New("RetrievalPlan channel score thresholds must be finite and non-negative")
	//	}
	//}
	return nil
}

// requirePositiveIDs 校验 ID 列表非空且均为正数
func requirePositiveIDs(values []int64, field string) error {
	if len(values) == 0 {
		return fmt.Errorf("RetrievalPlan %s must contain positive IDs", field)
	}
	for _, v := range values {
		if v <= 0 {
			return fmt.Errorf("RetrievalPlan %s must contain positive IDs", field)
		}
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
