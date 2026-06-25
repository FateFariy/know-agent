package vo

type DocumentChunkSourceType = int

const (
	ChunkSourceTypeOriginal DocumentChunkSourceType = iota + 1 // 原文切块
	ChunkSourceTypeEnriched                                    // 后处理补全文本
)

func DocumentChunkSourceTypeName(sourceType DocumentChunkSourceType) string {
	switch sourceType {
	case ChunkSourceTypeOriginal:
		return "原文切块"
	case ChunkSourceTypeEnriched:
		return "后处理补全文本"
	default:
		return "未知"
	}
}
