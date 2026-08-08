package entity

import (
	"github.com/swiftbit/know-agent/internal/domain/document/model/enum"
)

// DocumentParentBlock 文档父块实体
type DocumentParentBlock struct {
	ID                int64  `gorm:"column:id"`                  // 主键ID
	DocumentId        int64  `gorm:"column:document_id"`         // 文档ID
	TaskId            int64  `gorm:"column:task_id"`             // 任务ID
	PlanId            int64  `gorm:"column:plan_id"`             // 方案ID
	ParentNo          int    `gorm:"column:parent_no"`           // 父块序号
	SourceType        int    `gorm:"column:source_type"`         // 来源类型
	SectionPath       string `gorm:"column:section_path"`        // 章节路径
	StructureNodeId   int64  `gorm:"column:structure_node_id"`   // 结构节点ID
	StructureNodeType int    `gorm:"column:structure_node_type"` // 结构节点类型
	CanonicalPath     string `gorm:"column:canonical_path"`      // 规范路径
	ItemIndex         int    `gorm:"column:item_index"`          // 条目索引
	ParentText        string `gorm:"column:parent_text"`         // 父块文本内容
	CharCount         int    `gorm:"column:char_count"`          // 字符数
	TokenCount        int    `gorm:"column:token_count"`         // Token数量
	ChildCount        int    `gorm:"column:child_count"`         // 子块数量
	StartChunkNo      int    `gorm:"column:start_chunk_no"`      // 起始块序号
	EndChunkNo        int    `gorm:"column:end_chunk_no"`        // 结束块序号
	SourceTypeName    string `gorm:"-"`                          // 来源类型名称
}

func (d *DocumentParentBlock) FillEnumName() {
	d.SourceTypeName = enum.DocumentChunkSourceTypeName(d.SourceType)
}

//type DocumentParentBlocks []*DocumentParentBlock
//
//// CleanupAndUnique 过滤空文本并按 路径+序号+文本 去重
//func (b DocumentParentBlocks) CleanupAndUnique() DocumentParentBlocks {
//	seen := make(map[string]struct{})
//	result := make(DocumentParentBlocks, 0, len(b))
//	for _, block := range b {
//		if block == nil {
//			continue
//		}
//
//		trim := strutil.Trim(block.Text)
//		if trim != "" {
//			path := utils.BlankToDefault(block.CanonicalPath, block.SectionPath)
//			uniqueKey := fmt.Sprintf("%s||%d||%s", path, block.ItemIndex, trim)
//			if _, ok := seen[uniqueKey]; !ok {
//				seen[uniqueKey] = struct{}{}
//				result = append(result, CopyChunkCandidate(block, trim))
//			}
//		}
//	}
//
//	return result
//}
