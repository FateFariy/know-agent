package entity

// ParseArtifact 解析产物
type ParseArtifact struct {
	DocumentID    int64  `gorm:"column:document_id"`
	TaskID        int64  `gorm:"column:task_id"`
	ArtifactType  string `gorm:"column:artifact_type"` // 产物类型
	ObjectName    string `gorm:"column:object_name"`
	ContentHash   string `gorm:"column:content_hash"`   // 内容哈希值
	ParserName    string `gorm:"column:parser_name"`    // 解析器名称
	ParserVersion string `gorm:"column:parser_version"` // 解析器版本
	FileName      string `gorm:"-"`                     // 文件名
	ContentType   string `gorm:"-"`                     // 内容类型
	ContentBase64 string `gorm:"-"`                     // Base64编码内容
}
