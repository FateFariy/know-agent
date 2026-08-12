package model

import (
	"github.com/swiftbit/know-agent/common"
)

// ChatExchange 对话记录表
type ChatExchange struct {
	common.Model
	ConversationId                 string           `gorm:"column:conversation_id;type:varchar(64)"`               // 会话ID
	Question                       string           `gorm:"column:user_prompt;type:text"`                          // 用户提问
	Answer                         string           `gorm:"column:reply_content;type:text"`                        // 回复内容
	TurnStatus                     int              `gorm:"column:exchange_state;type:tinyint"`                    // 交互状态
	ThinkingSteps                  common.JSONArray `gorm:"column:thinking_steps;type:json"`                       // 思维步骤
	References                     common.JSONArray `gorm:"column:references;type:json"`                           // 参考列表
	Recommendations                common.JSONArray `gorm:"column:recommendations;type:json"`                      // 推荐问题列表
	UsedTools                      common.JSONArray `gorm:"column:used_tools;type:json"`                           // 工具使用列表
	DebugTrace                     string           `gorm:"column:debug_trace_json"`                               // 调试跟踪JSON
	ErrorMessage                   string           `gorm:"column:finish_note;type:varchar(500)"`                  // 错误信息
	FirstResponseTimeMs            int64            `gorm:"column:first_token_latency_ms;type:bigint"`             // 首包响应耗时(ms)
	TotalResponseTimeMs            int64            `gorm:"column:total_latency_ms;type:bigint"`                   // 总响应耗时(ms)
	KnowledgeBaseSelectionMode     string           `gorm:"column:knowledge_base_selection_mode;type:varchar(50)"` // 知识库选择模式
	SelectedKnowledgeBaseIdsJson   string           `gorm:"column:selected_knowledge_base_ids_json;type:json"`     // 选中知识库ID列表JSON
	SelectedKnowledgeBaseNamesJson string           `gorm:"column:selected_knowledge_base_names_json;type:json"`   // 选中知识库名称列表JSON
	RetrievalConfigSnapshot        string           `gorm:"column:retrieval_config_snapshot_json;type:json	"`      // 检索配置快照JSON
}
