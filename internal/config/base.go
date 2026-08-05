package config

import (
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/rest"
)

// Config 项目顶层全局配置
type Config struct {
	Http             rest.RestConf
	Auth             AuthConf
	Mysql            MysqlConf
	Redis            cache.CacheConf
	Minio            MinioConf
	Neo4j            Neo4jConf
	MQ               MQConf
	StructureParsing StructureParsingConf
	Embedding        EmbeddingConf
	Chunk            ChunkConf
	Milvus           MilvusConf
	Chat             ChatConf
	ChatModel        map[string]*LLMConf
}

// AuthConf 鉴权配置
type AuthConf struct {
	AccessSecret string
	AccessExpire time.Duration
}
