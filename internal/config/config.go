package config

import (
	"github.com/swiftbit/know-agent/common"
)

type Config struct {
	common.BaseConfig
	Minio MinioConf
	Neo4j Neo4jConf
	MQ    MQConf
}
type MinioConf struct {
	Endpoint         string `json:",omitempty,default=http://127.0.0.1:9000"`
	AccessKeyID      string `json:",omitempty,default=minioadmin"`
	SecretAccessKey  string `json:",omitempty,default=minioadmin"`
	BucketName       string `json:",omitempty,default=super-agent-document"`
	ObjectPrefix     string `json:",omitempty,default=rag/document"`
	ParsedTextPrefix string `json:",omitempty,default=rag/parsed-text"`
	UseSSL           bool   `json:",omitempty,default=false"`
}

type Neo4jConf struct {
	Enabled             bool   `json:",omitempty,default=false"`
	Uri                 string `json:",omitempty,default=bolt://127.0.0.1:7687"`
	Username            string `json:",omitempty,default=neo4j"`
	Password            string `json:",omitempty,default=neo4j"`
	Database            string `json:",omitempty,default=neo4j"`
	QueryTimeoutSeconds int    `json:",omitempty,default=5"`
}

type MQConf struct {
	ParseTopic string `json:",omitempty,default=know-agent-document"`
	IndexTopic string `json:",omitempty,default=know-agent-index"`
	Enabled    bool   `json:",omitempty,default=false"`
}

func (c Config) GetBaseConfig() *common.BaseConfig {
	return &c.BaseConfig
}
