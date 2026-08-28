package model

import (
	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ChatExchange 对话记录表
type ChatExchange struct {
	common.Model
	ConversationId             string                `gorm:"column:conversation_id;type:varchar(64)"`                             // 会话ID
	Question                   string                `gorm:"column:question;type:text"`                                           // 用户提问
	Answer                     string                `gorm:"column:answer;type:text"`                                             // 回复内容
	TurnStatus                 int                   `gorm:"column:turn_status;type:tinyint"`                                     // 交互状态
	ThinkingSteps              []string              `gorm:"column:thinking_steps;type:json;serializer:json"`                     // 思维步骤
	References                 []*vo.SearchReference `gorm:"column:references;type:json;serializer:json"`                         // 参考列表
	Recommendations            []string              `gorm:"column:recommendations;type:json;serializer:json"`                    // 建议列表
	UsedTools                  []string              `gorm:"column:used_tools;type:json;serializer:json"`                         // 工具使用列表
	DebugTrace                 string                `gorm:"column:debug_trace_json"`                                             // 调试跟踪JSON
	ErrorMessage               string                `gorm:"column:error_message;type:varchar(500)"`                              // 错误信息
	FirstResponseTimeMs        int64                 `gorm:"column:first_response_time_ms;type:bigint"`                           // 首包响应耗时(ms)
	TotalResponseTimeMs        int64                 `gorm:"column:total_response_time_ms;type:bigint"`                           // 总响应耗时(ms)
	KnowledgeBaseSelectionMode string                `gorm:"column:knowledge_base_selection_mode;type:varchar(50)"`               // 知识库选择模式
	SelectedKnowledgeBaseIds   []int64               `gorm:"column:selected_knowledge_base_ids_json;type:json;serializer:json"`   // 选中知识库ID列表JSON
	SelectedKnowledgeBaseNames []string              `gorm:"column:selected_knowledge_base_names_json;type:json;serializer:json"` // 选中知识库名称列表JSON
	RetrievalConfigSnapshot    string                `gorm:"column:retrieval_config_snapshot_json;type:json;"`                    // 检索配置快照JSON
}
