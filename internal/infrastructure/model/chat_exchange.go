package model

import "github.com/swiftbit/know-agent/common"

// ChatExchange 对话记录表
type ChatExchange struct {
	common.AuditModel
	ID                  int64  `gorm:"column:id;primaryKey"`          // 对话ID
	ConversationId      string `gorm:"column:conversation_id"`        // 会话ID
	Question            string `gorm:"column:question"`               // 用户问题
	Answer              string `gorm:"column:answer"`                 // 回答内容
	ThinkingSteps       string `gorm:"column:thinking_steps"`         // 思维步骤
	ReferenceList       string `gorm:"column:reference_list"`         // 参考列表
	RecommendationList  string `gorm:"column:recommendation_list"`    // 建议列表
	UsedToolList        string `gorm:"column:used_tool_list"`         // 工具使用列表
	DebugTraceJson      string `gorm:"column:debug_trace_json"`       // 调试跟踪JSON
	TurnStatus          int    `gorm:"column:turn_status"`            // 轮次状态
	ErrorMessage        string `gorm:"column:error_message"`          // 错误信息
	FirstResponseTimeMs int64  `gorm:"column:first_response_time_ms"` // 首个响应时间毫秒
	TotalResponseTimeMs int64  `gorm:"column:total_response_time_ms"` // 总响应时间毫秒
}
