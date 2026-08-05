package config

type MysqlConf struct {
	Endpoint string `json:",default=127.0.0.1:3306"`
	Username string
	Password string
	DbName   string
}

type Neo4jConf struct {
	Enabled             bool   `json:",default=false"`
	Uri                 string `json:",default=bolt://127.0.0.1:7687"`
	Username            string `json:",default=neo4j"`
	Password            string `json:",default=neo4j"`
	Database            string `json:",default=neo4j"`
	QueryTimeoutSeconds int    `json:",default=5"`
}

type MilvusConf struct {
	Addr       string `json:",default=127.0.0.1:19530"`
	Username   string `json:",optional"`
	Password   string `json:",optional"`
	Collection string
	MetricType string `json:",default=COSINE"`
}
