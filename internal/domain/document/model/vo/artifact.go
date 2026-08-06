package vo

// ParseArtifact 解析产物
type ParseArtifact struct {
	ArtifactType  string // 产物类型
	FileName      string // 文件名
	ContentType   string // 内容类型
	ContentBase64 string // Base64编码内容
	ContentHash   string // 内容哈希值
	ParserName    string // 解析器名称
	ParserVersion string // 解析器版本
}
