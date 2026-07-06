package vo

import (
	"time"
	"unicode/utf8"

	"github.com/duke-git/lancet/v2/convertor"

	"github.com/swiftbit/know-agent/common/utils"
)

type ChatRetrievalResult struct {
	ID               int64     `gorm:"column:id"`                 // 主键ID
	ConversationId   string    `gorm:"column:conversation_id"`    // 对话ID
	ExchangeId       int64     `gorm:"column:exchange_id"`        // 交互ID
	TraceId          string    `gorm:"column:trace_id"`           // 跟踪ID
	SubQuestionIndex int       `gorm:"column:sub_question_index"` // 子问题索引
	SubQuestion      string    `gorm:"column:sub_question"`       // 子问题
	ChannelType      string    `gorm:"column:channel_type"`       // 渠道类型
	ChannelRank      int       `gorm:"column:channel_rank"`       // 渠道排名
	RrfRank          int       `gorm:"column:rrf_rank"`           // RRF排名
	FinalRank        int       `gorm:"column:final_rank"`         // 最终排名
	OriginalScore    float64   `gorm:"column:original_score"`     // 原始分数
	RrfScore         float64   `gorm:"column:rrf_score"`          // RRF分数
	RerankScore      float64   `gorm:"column:rerank_score"`       // 重排分数
	GatePassed       int       `gorm:"column:gate_passed"`        // 是否通过门控
	IsElevated       int       `gorm:"column:is_elevated"`        // 是否升级
	IsSelected       int       `gorm:"column:is_selected"`        // 是否选中
	SelectionReason  string    `gorm:"column:selection_reason"`   // 选中原因
	DocumentId       int64     `gorm:"column:document_id"`        // 文档ID
	DocumentName     string    `gorm:"column:document_name"`      // 文档名称
	ChunkId          int64     `gorm:"column:chunk_id"`           // 分块ID
	ChunkNo          int       `gorm:"column:chunk_no"`           // 分块序号
	ParentBlockId    int64     `gorm:"column:parent_block_id"`    // 父块ID
	ParentBlockNo    int       `gorm:"column:parent_block_no"`    // 父块序号
	SectionPath      string    `gorm:"column:section_path"`       // 章节路径
	ChunkTextPreview string    `gorm:"column:chunk_text_preview"` // 分块文本预览
	ChunkCharCount   int       `gorm:"column:chunk_char_count"`   // 分块字符数
	CreateTime       time.Time `gorm:"column:create_time"`        // 创建时间
}

// SetDocumentInfo 设置文档基本信息
func (v *ChatRetrievalResult) SetDocumentInfo(doc *DocumentChunk) {
	if doc == nil {
		return
	}

	v.RrfScore = doc.RRFScore
	v.RerankScore = doc.RerankScore
	v.DocumentId = doc.DocumentId
	v.DocumentName = doc.DocumentName
	v.ChunkId, _ = convertor.ToInt(doc.ID)
	v.ChunkNo = doc.ChunkNo
	v.ParentBlockId, _ = convertor.ToInt(doc.ParentBlockId)
	v.ParentBlockNo = doc.ParentBlockNo
	v.SectionPath = doc.SectionPath
	v.IsElevated = doc.IsElevated

	// 设置原始分数
	v.OriginalScore = doc.Score

	// 设置文本预览
	v.ChunkCharCount = utf8.RuneCountInString(doc.OriginalSnippet)
	v.ChunkTextPreview = utils.ClipHead(doc.OriginalSnippet, 500)
}
