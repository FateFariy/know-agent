package model

import "github.com/swiftbit/know-agent/common"

type ChatRetrievalResult struct {
	common.Model
	ConversationId         string  `gorm:"column:conversation_id;type:varchar(64)"`        // 对话ID
	ExchangeId             int64   `gorm:"column:exchange_id;type:bigint"`                 // 交互ID
	TraceId                string  `gorm:"column:trace_id;type:varchar(64)"`               // 追踪ID
	SubQuestionIndex       int     `gorm:"column:sub_question_index;type:int"`             // 子问题索引
	SubQuestion            string  `gorm:"column:sub_question;type:text"`                  // 子问题
	CandidateId            string  `gorm:"column:candidate_id;type:varchar(64)"`           // 候选ID
	ChannelType            string  `gorm:"column:channel_type;type:varchar(50)"`           // 通道类型
	ChannelRank            int     `gorm:"column:channel_rank;type:int"`                   // 通道排名
	RrfRank                int     `gorm:"column:rrf_rank;type:int"`                       // RRF排名
	FinalRank              int     `gorm:"column:final_rank;type:int"`                     // 最终排名
	OriginalScore          float64 `gorm:"column:original_score;type:decimal(10,4)"`       // 原始分数
	RrfScore               float64 `gorm:"column:rrf_score;type:decimal(10,4)"`            // RRF分数
	HybridScore            float64 `gorm:"column:hybrid_score;type:decimal(10,4)"`         // 混合分数
	MetadataBoost          float64 `gorm:"column:metadata_boost;type:decimal(10,4)"`       // 元数据增强分数
	VectorScore            float64 `gorm:"column:vector_score;type:decimal(10,4)"`         // 向量分数
	KeywordScore           float64 `gorm:"column:keyword_score;type:decimal(10,4)"`        // 关键词分数
	RerankScore            float64 `gorm:"column:rerank_score;type:decimal(10,4)"`         // 重排分数
	GatePassed             int     `gorm:"column:gate_passed;type:tinyint"`                // 是否通过门控
	IsElevated             int     `gorm:"column:is_elevated;type:tinyint"`                // 是否提升
	IsSelected             int     `gorm:"column:is_selected;type:tinyint"`                // 是否被选中
	SelectionReason        string  `gorm:"column:selection_reason;type:varchar(500)"`      // 选中原因
	FilteredReason         string  `gorm:"column:filtered_reason;type:varchar(500)"`       // 过滤原因
	RankFeature            string  `gorm:"column:rank_feature;type:text"`                  // 排序特征
	DocumentId             int64   `gorm:"column:document_id;type:bigint"`                 // 文档ID
	DocumentName           string  `gorm:"column:document_name;type:varchar(255)"`         // 文档名称
	ChunkId                int64   `gorm:"column:chunk_id;type:bigint"`                    // 文本块ID
	ChunkType              string  `gorm:"column:chunk_type;type:varchar(50)"`             // 文本块类型
	ChunkNo                int     `gorm:"column:chunk_no;type:int"`                       // 文本块序号
	ParentBlockId          int64   `gorm:"column:parent_block_id;type:bigint"`             // 父块ID
	ParentBlockNo          int     `gorm:"column:parent_block_no;type:int"`                // 父块序号
	SectionPath            string  `gorm:"column:section_path;type:varchar(500)"`          // 章节路径
	ChunkTextPreview       string  `gorm:"column:chunk_text_preview;type:text"`            // 文本块预览
	ChunkCharCount         int     `gorm:"column:chunk_char_count;type:int"`               // 文本块字符数
	ContextIdentity        string  `gorm:"column:context_identity;type:varchar(255)"`      // 上下文标识
	CitationIdentity       string  `gorm:"column:citation_identity;type:varchar(255)"`     // 引用标识
	CitationIdentityHash   string  `gorm:"column:citation_identity_hash;type:varchar(64)"` // 引用标识哈希
	CitationEvidenceType   string  `gorm:"column:citation_evidence_type;type:varchar(50)"` // 引用证据类型
	ContextOnly            int     `gorm:"column:context_only;type:tinyint"`               // 仅上下文
	SourceEvidenceResolved int     `gorm:"column:source_evidence_resolved;type:tinyint"`   // 源证据已解析
}
