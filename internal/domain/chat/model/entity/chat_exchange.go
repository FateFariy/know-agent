package entity

import (
	"time"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/enum"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChatExchange 对话记录表
type ChatExchange struct {
	ID                             int64            `gorm:"column:id;primaryKey"`                                // 对话ID
	ConversationId                 string           `gorm:"column:conversation_id"`                              // 会话ID
	Question                       string           `gorm:"column:question"`                                     // 用户问题
	Answer                         string           `gorm:"column:answer"`                                       // 回答内容
	ThinkingSteps                  common.JSONArray `gorm:"column:thinking_steps;type:json"`                     // 思维步骤
	References                     common.JSONArray `gorm:"column:references;type:json"`                         // 参考列表
	Recommendations                common.JSONArray `gorm:"column:recommendations;type:json"`                    // 建议列表
	UsedTools                      common.JSONArray `gorm:"column:used_tools;type:json"`                         // 工具使用列表
	DebugTrace                     string           `gorm:"column:debug_trace_json"`                             // 调试跟踪JSON
	TurnStatus                     int              `gorm:"column:turn_status"`                                  // 轮次状态
	ErrorMessage                   string           `gorm:"column:error_message"`                                // 错误信息
	FirstResponseTimeMs            int64            `gorm:"column:first_response_time_ms"`                       // 首个响应时间毫秒
	TotalResponseTimeMs            int64            `gorm:"column:total_response_time_ms"`                       // 总响应时间毫秒
	KnowledgeBaseSelectionMode     string           `gorm:"column:knowledge_base_selection_mode"`                // 知识库选择模式
	SelectedKnowledgeBaseIdsJson   string           `gorm:"column:selected_knowledge_base_ids_json;type:json"`   // 选中知识库ID列表JSON
	SelectedKnowledgeBaseNamesJson string           `gorm:"column:selected_knowledge_base_names_json;type:json"` // 选中知识库名称列表JSON
	RetrievalConfigSnapshot        string           `gorm:"column:retrieval_config_snapshot_json;type:json"`     // 检索配置快照JSON
	CreateTime                     time.Time        `gorm:"column:create_time"`                                  // 创建时间
	UpdateTime                     time.Time        `gorm:"column:update_time"`                                  // 更新时间
}

// IsCompleted 判断对话轮次是否已完成
func (e *ChatExchange) IsCompleted() bool {
	if e == nil {
		return false
	}
	return e.TurnStatus == 0 || e.TurnStatus == enum.ChatTurnStatusCompleted
}

func (e *ChatExchange) ParseReferences() []*vo.SearchReference {
	var refs = make([]*vo.SearchReference, 0, len(e.References))
	for _, ref := range e.References {
		switch v := ref.(type) {
		case vo.SearchReference:
			refs = append(refs, &v)
		case *vo.SearchReference:
			refs = append(refs, v)
		}
	}
	return refs
}
