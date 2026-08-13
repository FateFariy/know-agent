package config

import "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"

// GlobalRagRuntimeConfigProvider 提供全局默认 RAG 配置
type GlobalRagRuntimeConfigProvider interface {
	CurrentOptions() *vo.RagRuntimeOptions
}
