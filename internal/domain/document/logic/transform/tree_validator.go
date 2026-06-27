package transform

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

type TreeValidator struct{}

func NewTreeValidator() *TreeValidator {
	return &TreeValidator{}
}

func (v *TreeValidator) Transform(documentTitle string, drafts []*vo.DocumentStructureNodeDraft, opts ...TransformerOption) []*vo.DocumentStructureNodeCandidate {
	if len(drafts) == 0 {
		return nil
	}

	draftMap := make(map[int]*vo.DocumentStructureNodeDraft)
	for _, draft := range drafts {
		if draft != nil && draft.NodeNo != 0 {
			draftMap[draft.NodeNo] = draft
		}
	}

	v.collapseSyntheticTitleSection(documentTitle, draftMap)
	v.repairNumberedHierarchy(draftMap)
	v.repairInvalidParents(draftMap)
	v.recomputeDepths(draftMap)
	v.rebuildPaths(documentTitle, draftMap)
	v.rebuildSiblingLinks(draftMap)

	nodeNos := make([]int, 0, len(draftMap))
	for nodeNo := range draftMap {
		nodeNos = append(nodeNos, nodeNo)
	}

	slices.SortFunc(nodeNos, func(a, b int) int { return a - b })

	result := make([]*vo.DocumentStructureNodeCandidate, 0, len(nodeNos))
	for _, nodeNo := range nodeNos {
		result = append(result, v.toCandidate(draftMap[nodeNo]))
	}

	return result
}

func (v *TreeValidator) collapseSyntheticTitleSection(documentTitle string, draftMap map[int]*vo.DocumentStructureNodeDraft) {
	normalizedTitle := support.NormalizeComparableTitle(documentTitle)
	if normalizedTitle == "" {
		return
	}

	var duplicateNodeNo int
	for _, draft := range draftMap {
		if draft.NodeNo == 1 || !draft.IsSection() || draft.ParentNodeNo != 1 || strutil.IsBlank(draft.NodeCode) {
			continue
		}

		if normalizedTitle == support.NormalizeComparableTitle(draft.Title) {
			duplicateNodeNo = draft.NodeNo
			break
		}
	}

	if duplicateNodeNo == 0 {
		return
	}

	for _, draft := range draftMap {
		if draft.ParentNodeNo != 0 && draft.ParentNodeNo == duplicateNodeNo {
			draft.ParentNodeNo = 1
		}
	}

	delete(draftMap, duplicateNodeNo)
}

func (v *TreeValidator) repairNumberedHierarchy(draftMap map[int]*vo.DocumentStructureNodeDraft) {
	numericPathMap := make(map[string]int)
	for _, draft := range draftMap {
		if !draft.IsSection() {
			continue
		}
		key := support.NumericKey(draft.NumericPath)
		if key != "" {
			if _, ok := numericPathMap[key]; !ok {
				numericPathMap[key] = draft.NodeNo
			}
		}
	}

	for _, draft := range draftMap {
		if !draft.IsSection() {
			continue
		}

		numericPath := draft.NumericPath
		if numericPath == nil || len(numericPath) == 0 {
			continue
		}

		if len(numericPath) == 1 {
			draft.ParentNodeNo = intPtr(1)
			continue
		}

		directParentKey := support.NumericKey(numericPath[:len(numericPath)-1])
		if directParent, ok := numericPathMap[directParentKey]; ok {
			draft.ParentNodeNo = &directParent
			continue
		}

		chapterParentKey := support.NumericKey([]int{numericPath[0]})
		if chapterParent, ok := numericPathMap[chapterParentKey]; ok {
			draft.ParentNodeNo = &chapterParent
		}
	}
}

func (v *TreeValidator) repairInvalidParents(draftMap map[int]*vo.DocumentStructureNodeDraft) {
	for _, draft := range draftMap {
		if draft == nil || draft.NodeNo == 1 {
			continue
		}

		var parent *vo.DocumentStructureNodeDraft
		if draft.ParentNodeNo != nil {
			parent = draftMap[*draft.ParentNodeNo]
		}

		if parent == nil {
			draft.ParentNodeNo = intPtr(1)
			continue
		}

		if draft.IsSection() && parent.IsListLike() {
			if parent.ParentNodeNo != nil {
				draft.ParentNodeNo = parent.ParentNodeNo
			} else {
				draft.ParentNodeNo = intPtr(1)
			}
		}
	}
}

func (v *TreeValidator) recomputeDepths(draftMap map[int]*vo.DocumentStructureNodeDraft) {
	root := draftMap[1]
	if root == nil {
		return
	}
	root.Depth = 0

	nodeNos := make([]int, 0, len(draftMap))
	for nodeNo := range draftMap {
		nodeNos = append(nodeNos, nodeNo)
	}

	for i := 0; i < len(nodeNos)-1; i++ {
		for j := i + 1; j < len(nodeNos); j++ {
			if nodeNos[i] > nodeNos[j] {
				nodeNos[i], nodeNos[j] = nodeNos[j], nodeNos[i]
			}
		}
	}

	for _, nodeNo := range nodeNos {
		draft := draftMap[nodeNo]
		if draft == nil || draft.NodeNo == 1 {
			continue
		}

		var parent *vo.DocumentStructureNodeDraft
		if draft.ParentNodeNo != nil {
			parent = draftMap[*draft.ParentNodeNo]
		}

		if parent != nil {
			draft.Depth = parent.Depth + 1
		} else {
			draft.Depth = 1
		}
	}
}

func (v *TreeValidator) rebuildPaths(documentTitle string, draftMap map[int]*vo.DocumentStructureNodeDraft) {
	for _, draft := range draftMap {
		if draft == nil {
			continue
		}

		if draft.NodeNo == 1 {
			draft.CanonicalPath = "/document"
			draft.SectionPath = ""
			continue
		}

		var parent *vo.DocumentStructureNodeDraft
		if draft.ParentNodeNo != nil {
			parent = draftMap[*draft.ParentNodeNo]
		}

		parentCanonicalPath := "/document"
		if parent != nil && parent.CanonicalPath != "" {
			parentCanonicalPath = parent.CanonicalPath
		}

		parentSectionPath := ""
		if parent != nil && parent.SectionPath != "" {
			parentSectionPath = parent.SectionPath
		}

		segment := v.buildPathSegment(draft)
		draft.CanonicalPath = parentCanonicalPath + "/" + segment

		if draft.IsSection() {
			draft.SectionPath = v.joinSectionPath(parentSectionPath, v.displayTitle(draft))
		} else {
			draft.SectionPath = parentSectionPath
		}
	}
}

func (v *TreeValidator) rebuildSiblingLinks(draftMap map[int]*vo.DocumentStructureNodeDraft) {
	childrenByParent := make(map[int][]*vo.DocumentStructureNodeDraft)
	for _, draft := range draftMap {
		if draft == nil || draft.NodeNo == 1 {
			continue
		}

		var parentNodeNo int
		if draft.ParentNodeNo != nil {
			parentNodeNo = *draft.ParentNodeNo
		} else {
			parentNodeNo = 1
		}

		childrenByParent[parentNodeNo] = append(childrenByParent[parentNodeNo], draft)
	}

	for _, siblings := range childrenByParent {
		for i := 0; i < len(siblings)-1; i++ {
			for j := i + 1; j < len(siblings); j++ {
				if siblings[i].LineNo > siblings[j].LineNo {
					siblings[i], siblings[j] = siblings[j], siblings[i]
				}
			}
		}

		for index := 0; index < len(siblings); index++ {
			current := siblings[index]

			if index == 0 {
				current.PrevSiblingNodeNo = intPtr(0)
			} else {
				prevNo := siblings[index-1].NodeNo
				current.PrevSiblingNodeNo = &prevNo
			}

			if index == len(siblings)-1 {
				current.NextSiblingNodeNo = intPtr(0)
			} else {
				nextNo := siblings[index+1].NodeNo
				current.NextSiblingNodeNo = &nextNo
			}
		}
	}
}

func (v *TreeValidator) toCandidate(draft *vo.DocumentStructureNodeDraft) *vo.DocumentStructureNodeCandidate {
	return &vo.DocumentStructureNodeCandidate{
		NodeNo:            draft.NodeNo,
		NodeType:          draft.NodeType,
		ParentNodeNo:      draft.ParentNodeNo,
		PrevSiblingNodeNo: draft.PrevSiblingNodeNo,
		NextSiblingNodeNo: draft.NextSiblingNodeNo,
		Depth:             draft.Depth,
		NodeCode:          draft.NodeCode,
		Title:             draft.Title,
		AnchorText:        draft.AnchorText,
		CanonicalPath:     draft.CanonicalPath,
		SectionPath:       draft.SectionPath,
		ContentText:       draft.ContentText.String(),
		ItemIndex:         draft.ItemIndex,
	}
}

func (v *TreeValidator) joinSectionPath(parentSectionPath, currentTitle string) string {
	if parentSectionPath == "" {
		if currentTitle == "" {
			return ""
		}
		return currentTitle
	}
	if currentTitle == "" {
		return parentSectionPath
	}
	return parentSectionPath + " > " + currentTitle
}

func (v *TreeValidator) buildPathSegment(draft *vo.DocumentStructureNodeDraft) string {
	if draft == nil {
		return "node"
	}

	if draft.IsListLike() {
		if draft.ItemIndex > 0 {
			return fmt.Sprintf("item-%d", draft.ItemIndex)
		}
		return v.slug(v.displayTitle(draft))
	}

	code := strings.TrimSpace(draft.NodeCode)
	if code != "" {
		return v.slug(code)
	}

	return v.slug(v.displayTitle(draft))
}

func (v *TreeValidator) displayTitle(draft *vo.DocumentStructureNodeDraft) string {
	code := strings.TrimSpace(draft.NodeCode)
	title := strings.TrimSpace(draft.Title)

	if code == "" {
		return title
	}
	if strings.HasPrefix(title, code) {
		return title
	}
	return code + " " + title
}

func (v *TreeValidator) slug(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "node"
	}

	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, "-")
	normalized = regexp.MustCompile(`[^\p{L}\p{N}_.-]`).ReplaceAllString(normalized, "")

	if normalized == "" {
		return "node"
	}
	return normalized
}
