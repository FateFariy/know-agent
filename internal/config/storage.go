package config

type MinioConf struct {
	Addr                string `json:",default=127.0.0.1:9000"`
	AccessKeyID         string `json:",default=minioadmin"`
	SecretAccessKey     string `json:",default=minioadmin"`
	BucketName          string `json:",default=super-agent-document"`
	ObjectPrefix        string `json:",default=rag/document"`
	ParsedTextPrefix    string `json:",default=rag/parsed-text"`
	ParseArtifactPrefix string `json:",default=rag/parse-artifact"`
	UseSSL              bool   `json:",default=false"`
}

type MQConf struct {
	Addr       string `json:",default=127.0.0.1"`
	ParseTopic string `json:",default=know-agent-document"`
	IndexTopic string `json:",default=know-agent-index"`
	Retry      int    `json:",default=3"`
	Enabled    bool   `json:",default=false"`
}
