package vo

// ============================================================
// ChunkSourceType 块来源类型
// ============================================================

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

// ============================================================
// PipelineType 流水线类型
// ============================================================

type PipelineType = string

const (
	PipelineTypeParent PipelineType = "PARENT" // 父块
	PipelineTypeChild               = "CHILD"  // 子块
)

func PipelineTypeName(pt PipelineType) string {
	switch pt {
	case PipelineTypeParent:
		return "父块"
	case PipelineTypeChild:
		return "子块"
	default:
		return ""
	}
}

// ============================================================
// VectorStatus 向量状态
// ============================================================

type VectorStatus = int

const (
	VectorStatusWaitVector    VectorStatus = iota + 1 // 待向量化
	VectorStatusVectorizing                           // 向量化中
	VectorStatusVectorSuccess                         // 向量化成功
	VectorStatusVectorFailed                          // 向量化失败
)

func VectorStatusName(vs VectorStatus) string {
	switch vs {
	case VectorStatusWaitVector:
		return "待向量化"
	case VectorStatusVectorizing:
		return "向量化中"
	case VectorStatusVectorSuccess:
		return "向量化成功"
	case VectorStatusVectorFailed:
		return "向量化失败"
	default:
		return ""
	}
}

// ============================================================
// VectorStoreType 向量存储类型
// ============================================================

type VectorStoreType = int

const (
	VectorStoreTypeMilvus VectorStoreType = iota + 1
	VectorStoreTypePgVector
)

func VectorStoreTypeName(vs VectorStoreType) string {
	switch vs {
	case VectorStoreTypeMilvus:
		return "Milvus"
	case VectorStoreTypePgVector:
		return "PgVector"
	default:
		return ""
	}
}
