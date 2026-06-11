package vo

// ChunkSourceType 块来源类型
type ChunkSourceType = int

const (
	ChunkSourceTypeUnknown ChunkSourceType = iota
	ChunkSourceTypeText
	ChunkSourceTypeTable
	ChunkSourceTypeImage
)
