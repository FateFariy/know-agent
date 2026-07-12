package graph

import (
	"context"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/model/entity"
	docent "github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

// DefaultStructureGraphQuerier 默认结构图查询器。
// 直接从 document_structure_node 读取章节节点，供文档问答路由器使用。
type DefaultStructureGraphQuerier struct {
	db *gorm.DB
}

// NewDefaultStructureGraphQuerier 创建默认结构图查询器。
func NewDefaultStructureGraphQuerier(db *gorm.DB) *DefaultStructureGraphQuerier {
	return &DefaultStructureGraphQuerier{db: db}
}

var _ logic.StructureGraphQuerier = (*DefaultStructureGraphQuerier)(nil)

func (q *DefaultStructureGraphQuerier) ListSections(ctx context.Context, documentId int64) ([]*entity.GraphSection, error) {
	if documentId == 0 || q.db == nil {
		return nil, nil
	}
	var rows []*docent.DocumentStructureNode
	err := q.db.WithContext(ctx).
		Table("document_structure_node").
		Where("document_id = ?", documentId).
		Order("node_id ASC").
		Scan(&rows).Error
	if err != nil {
		logx.Errorf("ListSections 失败, documentId=%d, err=%v", documentId, err)
		return nil, err
	}
	result := make([]*entity.GraphSection, 0, len(rows))
	for _, r := range rows {
		result = append(result, toGraphSection(r))
	}
	return result, nil
}

func (q *DefaultStructureGraphQuerier) FindSectionById(ctx context.Context, documentId, nodeId int64) (*entity.GraphSection, error) {
	if documentId == 0 || nodeId == 0 || q.db == nil {
		return nil, nil
	}
	var r docent.DocumentStructureNode
	err := q.db.WithContext(ctx).
		Table("document_structure_node").
		Where("document_id = ? AND node_id = ?", documentId, nodeId).
		First(&r).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			logx.Errorf("FindSectionById 失败, documentId=%d, nodeId=%d, err=%v", documentId, nodeId, err)
		}
		return nil, err
	}
	return toGraphSection(&r), nil
}

func (q *DefaultStructureGraphQuerier) FindSectionByCode(ctx context.Context, documentId int64, sectionCode string) (*entity.GraphSection, error) {
	if documentId == 0 || q.db == nil || strutil.IsBlank(sectionCode) {
		return nil, nil
	}
	normalized := strings.TrimSpace(strings.ToLower(sectionCode))
	sections, err := q.ListSections(ctx, documentId)
	if err != nil || len(sections) == 0 {
		return nil, err
	}
	for _, s := range sections {
		if strings.EqualFold(strings.TrimSpace(s.NodeCode), normalized) {
			return s, nil
		}
	}
	return nil, nil
}

func (q *DefaultStructureGraphQuerier) FindBestSection(ctx context.Context, documentId int64, question, anchorHint string) (*entity.GraphSection, error) {
	sections, err := q.ListSections(ctx, documentId)
	if err != nil || len(sections) == 0 {
		return nil, err
	}
	question = strings.ToLower(strings.TrimSpace(question))
	anchorHint = strings.ToLower(strings.TrimSpace(anchorHint))
	if strutil.IsBlank(question) && strutil.IsBlank(anchorHint) {
		return nil, nil
	}
	var best *entity.GraphSection
	bestScore := 0.0
	for _, s := range sections {
		score := 0.0
		if strutil.IsNotBlank(s.Title) && strings.Contains(strings.ToLower(s.Title), question) {
			score += 90
		}
		if strutil.IsNotBlank(s.SectionPath) && strings.Contains(strings.ToLower(s.SectionPath), question) {
			score += 70
		}
		if strutil.IsNotBlank(s.AnchorText) && strings.Contains(strings.ToLower(s.AnchorText), question) {
			score += 60
		}
		if strutil.IsNotBlank(s.ContentText) && strings.Contains(strings.ToLower(s.ContentText), question) {
			score += 30
		}
		if strutil.IsNotBlank(anchorHint) &&
			(strings.Contains(strings.ToLower(s.Title), anchorHint) ||
				strings.Contains(strings.ToLower(s.SectionPath), anchorHint) ||
				strings.Contains(strings.ToLower(s.AnchorText), anchorHint)) {
			score += 25
		}
		if score > bestScore {
			bestScore = score
			best = s
		}
	}
	if bestScore < 30 {
		return nil, nil
	}
	return best, nil
}

func (q *DefaultStructureGraphQuerier) FindSectionWithChildren(ctx context.Context, documentId, sectionNodeId int64) (*entity.GraphSectionWithChildren, error) {
	section, err := q.FindSectionById(ctx, documentId, sectionNodeId)
	if err != nil || section == nil {
		return nil, err
	}
	if q.db == nil {
		return &entity.GraphSectionWithChildren{Section: section}, nil
	}
	var rows []*docent.DocumentStructureNode
	err = q.db.WithContext(ctx).
		Table("document_structure_node").
		Where("document_id = ? AND parent_node_id = ?", documentId, sectionNodeId).
		Order("node_id ASC").
		Scan(&rows).Error
	if err != nil {
		logx.Errorf("FindSectionWithChildren 失败, documentId=%d, sectionNodeId=%d, err=%v", documentId, sectionNodeId, err)
		return nil, err
	}
	children := make([]*entity.GraphSection, 0, len(rows))
	for _, r := range rows {
		children = append(children, toGraphSection(r))
	}
	return &entity.GraphSectionWithChildren{Section: section, Children: children}, nil
}

func (q *DefaultStructureGraphQuerier) FindSectionWithSiblings(ctx context.Context, documentId, sectionNodeId int64) (*entity.GraphSectionWithSiblings, error) {
	section, err := q.FindSectionById(ctx, documentId, sectionNodeId)
	if err != nil || section == nil {
		return nil, err
	}
	parent, _ := q.FindSectionById(ctx, documentId, section.ParentNodeId)
	if q.db == nil {
		return &entity.GraphSectionWithSiblings{Section: section, Parent: parent}, nil
	}
	var rows []*docent.DocumentStructureNode
	err = q.db.WithContext(ctx).
		Table("document_structure_node").
		Where("document_id = ? AND parent_node_id = ?", documentId, section.ParentNodeId).
		Order("node_id ASC").
		Scan(&rows).Error
	if err != nil {
		logx.Errorf("FindSectionWithSiblings 失败, documentId=%d, sectionNodeId=%d, err=%v", documentId, sectionNodeId, err)
		return nil, err
	}
	var previous, next *entity.GraphSection
	found := false
	for _, r := range rows {
		gs := toGraphSection(r)
		if r.NodeId == sectionNodeId {
			found = true
			continue
		}
		if !found {
			previous = gs
		} else if next == nil {
			next = gs
		}
	}
	return &entity.GraphSectionWithSiblings{
		Section:         section,
		Parent:          parent,
		PreviousSibling: previous,
		NextSibling:     next,
	}, nil
}

func (q *DefaultStructureGraphQuerier) BuildGraphResult(ctx context.Context, documentId, sectionNodeId int64, itemIndex *int, itemKeyword string) (*entity.GraphQueryResult, error) {
	withChildren, err := q.FindSectionWithChildren(ctx, documentId, sectionNodeId)
	if err != nil {
		return nil, err
	}
	withSiblings, err := q.FindSectionWithSiblings(ctx, documentId, sectionNodeId)
	if err != nil {
		return nil, err
	}
	result := &entity.GraphQueryResult{
		ItemIndex: itemIndex,
	}
	if withChildren != nil {
		result.TargetSection = withChildren.Section
		result.Children = withChildren.Children
	}
	if withSiblings != nil {
		if result.TargetSection == nil {
			result.TargetSection = withSiblings.Section
		}
		result.ParentSection = withSiblings.Parent
		result.PreviousSibling = withSiblings.PreviousSibling
		result.NextSibling = withSiblings.NextSibling
	}
	return result, nil
}

func toGraphSection(r *docent.DocumentStructureNode) *entity.GraphSection {
	if r == nil {
		return nil
	}
	return &entity.GraphSection{
		NodeId:       r.NodeId,
		DocumentId:   r.DocumentId,
		ParentNodeId: r.ParentNodeId,
		NodeCode:     r.NodeCode,
		Title:        r.Title,
		SectionPath:  r.SectionPath,
		AnchorText:   r.AnchorText,
		ContentText:  r.ContentText,
	}
}
