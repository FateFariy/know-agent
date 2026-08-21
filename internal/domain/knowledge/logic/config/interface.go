package config

import "github.com/swiftbit/know-agent/internal/domain/knowledge/model/vo"

// GlobalConfigProvider 提供全局默认配置
type GlobalConfigProvider interface {
	CurrentOptions() *vo.RagRuntimeOptions
	CurrentIndexingOptions() *vo.IndexingOptions
}
