package vo

// ChunkSourceType 块来源类型
type ChunkSourceType = int

const (
	ChunkSourceTypeUnknown ChunkSourceType = iota
	ChunkSourceTypeText
	ChunkSourceTypeTable
	ChunkSourceTypeImage
)

func ChunkSourceTypeName(cst ChunkSourceType) string {
	switch cst {
	case ChunkSourceTypeText:
		return "文本"
	case ChunkSourceTypeTable:
		return "表格"
	case ChunkSourceTypeImage:
		return "图片"
	default:
		return "未知"
	}
}
