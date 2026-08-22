package channel

import (
	"fmt"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/rag"
)

// DocumentRetrieve 文档检索
type DocumentRetrieve struct {
	Question          string                   `json:"question"`          // 问题
	RetrievalQuery    string                   `json:"retrievalQuery"`    // 检索查询
	DocumentId        int64                    `json:"documentId"`        // 文档ID
	TaskId            int64                    `json:"taskId"`            // 任务ID
	DocumentIds       []int64                  `json:"documentIds"`       // 文档ID列表
	TaskIds           []int64                  `json:"taskIds"`           // 任务ID列表
	TopK              int                      `json:"topK"`              // 返回数量
	Filters           *DocumentRetrieveFilters `json:"filters"`           // 过滤器
	QueryContextHints []string                 `json:"queryContextHints"` // 查询上下文提示
}

// DocumentRetrieveFilters 文档检索过滤器
type DocumentRetrieveFilters struct {
	DocumentNameHints    []string // 文档名称提示
	SectionPathHints     []string // 段落路径提示
	CanonicalPathHints   []string // 规范路径提示
	StructureNodeIdHints []int64  // 结构节点ID提示
	ItemIndexHints       []int    // 条目索引提示
	YearHints            []string // 年份提示
}

// NewDocumentRetrieve 根据检索执行请求和渠道名称构建文档检索请求。
func NewDocumentRetrieve(channelName string, input *rag.ExecutionInput) (*DocumentRetrieve, error) {
	if input == nil {
		return nil, fmt.Errorf("ExecutionInput is required for document retrieval")
	}

	// 获取指定渠道的配置
	channel, err := input.RequireChannel(channelName)
	if err != nil {
		return nil, err
	}

	req := &DocumentRetrieve{
		Question:          input.SubQuestion,
		RetrievalQuery:    input.SubQuestion,
		TopK:              channel.TopK,
		QueryContextHints: input.ContextHints,
		DocumentIds:       input.DocumentScope,
		TaskIds:           input.TaskScope,
	}

	// 解析单文档/单任务作用域，返回主 ID
	if len(input.DocumentScope) == 1 && len(input.TaskScope) == 1 {
		req.DocumentId, req.TaskId = input.DocumentScope[0], input.TaskScope[0]
	}

	// 设置过滤器
	if input.Filters != nil {
		req.Filters = &DocumentRetrieveFilters{
			DocumentNameHints: input.Filters.DocumentNameHints,
			SectionPathHints:  input.Filters.SectionPathHints,
			YearHints:         input.Filters.YearHints,
		}
	}

	return req, nil
}

func (d *DocumentRetrieveFilters) IsEmpty() bool {
	return d == nil && len(d.DocumentNameHints) == 0 &&
		len(d.SectionPathHints) == 0 &&
		len(d.CanonicalPathHints) == 0 &&
		len(d.ItemIndexHints) == 0 &&
		len(d.YearHints) == 0
}

func (d *DocumentRetrieve) Validate() bool {
	return d != nil && len(d.DocumentIds) > 0 && len(d.TaskIds) > 0 &&
		utils.IsNotBlank(d.Question) && utils.IsNotBlank(d.RetrievalQuery)
}
