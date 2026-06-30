package vo

// VectorStatus 向量状态
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
