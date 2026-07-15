package model

import "github.com/swiftbit/know-agent/common"

// ChatExchange 对话记录表
type ChatExchange struct {
	common.Model
	ConversationId      string           `gorm:"column:conversation_id"`        // 会话ID
	Question            string           `gorm:"column:question"`               // 用户问题
	Answer              string           `gorm:"column:answer"`                 // 回答内容
	ThinkingSteps       common.JSONArray `gorm:"column:thinking_steps"`         // 思维步骤
	References          common.JSONArray `gorm:"column:references"`             // 参考列表
	Recommendations     common.JSONArray `gorm:"column:recommendations"`        // 建议列表
	UsedTools           common.JSONArray `gorm:"column:used_tools"`             // 工具使用列表
	DebugTrace          string           `gorm:"column:debug_trace_json"`       // 调试跟踪JSON
	TurnStatus          int              `gorm:"column:turn_status"`            // 轮次状态
	ErrorMessage        string           `gorm:"column:error_message"`          // 错误信息
	FirstResponseTimeMs int64            `gorm:"column:first_response_time_ms"` // 首个响应时间毫秒
	TotalResponseTimeMs int64            `gorm:"column:total_response_time_ms"` // 总响应时间毫秒
}
