package transform

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

var (
	spaceToDashRegex = regexp.MustCompile(`\s+`)
	invalidCharRegex = regexp.MustCompile(`[^\p{L}\p{N}_.-]`)
)

type TreeValidator struct{}

func NewTreeValidator() *TreeValidator {
	return &TreeValidator{}
}

// Transform 执行整棵草稿树的规范化与修复流程，最终返回按节点编号排序的候选节点列表。
//
// 处理步骤：
//  1. 构建 NodeNo -> Draft 的映射表，过滤无效条目
//  2. 折叠与文档标题重复的合成章节（消除冗余顶层节点）
//  3. 依据 NumericPath 修复章节层级关系
//  4. 校正无效/非法的父节点引用
//  5. 依据最新层级重新计算各节点的深度
//  6. 重建 CanonicalPath / SectionPath（显示路径）
//  7. 重建兄弟节点之间的双向链表指针
//  8. 将 Draft 转换为 Candidate 并按 NodeNo 排序返回
func (v *TreeValidator) Transform(documentTitle string, drafts []*vo.DocumentStructureNodeDraft, opts ...TransformerOption) []*vo.DocumentStructureNodeCandidate {
	if len(drafts) == 0 {
		return nil
	}

	// 构建节点编号映射表（过滤无效条目）
	draftMap := make(map[int]*vo.DocumentStructureNodeDraft)
	for _, draft := range drafts {
		if draft != nil && draft.NodeNo != 0 {
			draftMap[draft.NodeNo] = draft
		}
	}

	// 折叠重复标题
	v.collapseSyntheticTitleSection(documentTitle, draftMap)

	// 修复章节层级关系
	v.repairNumberedHierarchy(draftMap)

	// 校正无效/非法的父节点引用
	v.repairInvalidParents(draftMap)

	// 依据最新层级重新计算各节点的深度
	v.recomputeDepths(draftMap)

	// 重建显示路径
	v.rebuildPaths(draftMap)

	// 重建兄弟节点之间的双向链表指针
	v.rebuildSiblingLinks(draftMap)

	// 转换为候选节点并按节点编号升序排序
	candidates := slice.Map(maputil.Values(draftMap), func(index int, draft *vo.DocumentStructureNodeDraft) *vo.DocumentStructureNodeCandidate {
		return draft.ToCandidate()
	})
	slices.SortFunc(candidates, func(a, b *vo.DocumentStructureNodeCandidate) int { return a.NodeNo - b.NodeNo })

	return candidates
}

// collapseSyntheticTitleSection 折叠与文档标题重复的合成章节节点，避免文档标题被重复作为一级章节。
//
// 处理步骤：
//  1. 规范化文档标题；若为空则直接返回
//  2. 查找一个满足以下条件的重复节点：是章节、父节点为根(1)、且拥有非空 NodeCode
//  3. 将该重复节点的所有子节点挂到根节点，然后删除该重复节点
func (v *TreeValidator) collapseSyntheticTitleSection(documentTitle string, draftMap map[int]*vo.DocumentStructureNodeDraft) {
	// 规范化文档标题（去除大小写/空白差异用于等值比较）
	normalizedTitle := support.NormalizeComparableTitle(documentTitle)
	if normalizedTitle == "" {
		return
	}

	// 定位与文档标题重复的章节节点（需为根的直属章节且带有节点编号非空）
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

	// 将该重复节点的子节点提升至根节点，然后删除重复节点
	for _, draft := range draftMap {
		if draft.ParentNodeNo != 0 && draft.ParentNodeNo == duplicateNodeNo {
			draft.ParentNodeNo = 1
		}
	}

	delete(draftMap, duplicateNodeNo)
}

// repairNumberedHierarchy 基于 NumericPath（例如 [1,2,3]）重新推断章节之间的父子关系。
//
// 处理步骤：
//  1. 建立 "数字路径键 -> 节点编号" 的索引，便于 O(1) 查找父级
//  2. 重新为每个章节赋值父节点：
//     - 长度为 1 → 根节点
//     - 否则尝试定位直接父节点（前缀路径）
//     - 找不到直接父节点时，退回到对应一级章节作为父节点
func (v *TreeValidator) repairNumberedHierarchy(draftMap map[int]*vo.DocumentStructureNodeDraft) {
	// 构建数字路径键 → 节点编号映射（仅章节类型参与）
	numericPathMap := make(map[string]int)
	for _, draft := range draftMap {
		if draft.IsSection() {
			key := support.NumericKey(draft.NumericPath)
			if key != "" {
				if _, ok := numericPathMap[key]; !ok {
					numericPathMap[key] = draft.NodeNo
				}
			}
		}
	}

	// 依据 NumericPath 长度逐步确定父节点
	for _, draft := range draftMap {
		if !draft.IsSection() {
			continue
		}

		numericPath := draft.NumericPath
		if len(numericPath) == 0 {
			continue
		}

		// 长度为 1 → 直接挂根
		if len(numericPath) == 1 {
			draft.ParentNodeNo = 1
			continue
		}

		// 优先匹配直接父节点（NumericPath 去掉最后一段）
		directParentKey := support.NumericKey(numericPath[:len(numericPath)-1])
		if directParent, ok := numericPathMap[directParentKey]; ok {
			draft.ParentNodeNo = directParent
			continue
		}

		// 回退策略：挂载到同级的一级章节（使用首位数字匹配）
		chapterParentKey := strconv.Itoa(numericPath[0])
		if chapterParent, ok := numericPathMap[chapterParentKey]; ok {
			draft.ParentNodeNo = chapterParent
		}
	}
}

// repairInvalidParents 修复无效的父子关系：确保章节节点不会挂到列表类节点下、且所有节点都有合法的父节点。

// 处理步骤：
//  1. 跳过根节点（NodeNo==1）
//  2. 如果当前父节点存在但"章节挂到列表类节点"下这种非法结构，则上提一层
//  3. 如果父节点不存在，则默认挂到根节点
func (v *TreeValidator) repairInvalidParents(draftMap map[int]*vo.DocumentStructureNodeDraft) {
	for _, draft := range draftMap {
		// 根节点无需处理
		if draft.NodeNo == 1 {
			continue
		}

		// 父节点存在且非法 → 将章节上提一层（避免章节成为列表的子节点）
		if parent, ok := draftMap[draft.ParentNodeNo]; ok {
			if draft.IsSection() && parent.IsListLike() {
				draft.ParentNodeNo = utils.Ternary(parent.ParentNodeNo != 0, parent.ParentNodeNo, 1)
			}
		} else {
			// 父节点缺失 → 默认挂到根节点
			draft.ParentNodeNo = 1
		}
	}
}

// recomputeDepths 依据最新的父节点关系重新计算每个节点的深度（Depth）。
//
// 处理步骤：
//  1. 根节点深度固定为 0
//  2. 按节点编号排序后依次计算其他节点的深度：父节点 Depth + 1；父节点不存在则默认为 1
func (v *TreeValidator) recomputeDepths(draftMap map[int]*vo.DocumentStructureNodeDraft) {
	// 根节点深度固定为 0
	root := draftMap[1]
	root.Depth = 0

	// 步骤2：按 NodeNo 升序遍历，确保父节点先于子节点被处理
	ordered := maputil.Values(draftMap)
	slices.SortFunc(ordered, func(a, b *vo.DocumentStructureNodeDraft) int { return a.NodeNo - b.NodeNo })
	for _, draft := range ordered {
		if draft.NodeNo != 1 {
			draft.Depth = 1
			if parent, ok := draftMap[draft.ParentNodeNo]; ok {
				draft.Depth = parent.Depth + 1
			}
		}
	}
}

// rebuildPaths 重建每个节点的 CanonicalPath（规范化 URL 路径）与 SectionPath（可读的章节显示路径）。
//
// 处理步骤：
//  1. 根节点固定为 "/document"，章节显示路径为空
//  2. 其他节点从父节点继承路径，并附加由 buildPathSegment 生成的段落
//  3. 章节类型节点会同时更新显示路径（使用 " > " 连接父章节路径与当前标题）
func (v *TreeValidator) rebuildPaths(draftMap map[int]*vo.DocumentStructureNodeDraft) {
	for _, draft := range draftMap {
		// 根节点特殊处理
		if draft.NodeNo == 1 {
			draft.CanonicalPath = "/document"
			draft.SectionPath = ""
			continue
		}

		// 获取父节点路径（含容错默认值）
		parentCanonicalPath := "/document"
		parentSectionPath := ""
		if parent, ok := draftMap[draft.ParentNodeNo]; ok {
			parentCanonicalPath = utils.BlankToDefault(parent.CanonicalPath, "/document")
			parentSectionPath = parent.SectionPath
		}

		// 构建当前节点的规范化路径片段并组装
		segment := v.buildPathSegment(draft)
		draft.CanonicalPath = parentCanonicalPath + "/" + segment

		// 章节类型节点同时更新章节显示路径；其他类型沿用上一级路径
		if draft.IsSection() {
			draft.SectionPath = v.joinSectionPath(parentSectionPath, v.displayTitle(draft))
		} else {
			draft.SectionPath = parentSectionPath
		}
	}
}

// rebuildSiblingLinks 重建兄弟节点间的双向指针（PrevSiblingNodeNo / NextSiblingNodeNo）
//
// 处理步骤：
//  1. 按父节点分组，收集所有子节点
//  2. 每组兄弟节点按 LineNo（原文行号）排序以保持文档顺序
//  3. 遍历每个兄弟列表，设置前驱与后继的节点编号
func (v *TreeValidator) rebuildSiblingLinks(draftMap map[int]*vo.DocumentStructureNodeDraft) {
	// 按父节点分组所有非根节点
	childrenByParent := make(map[int][]*vo.DocumentStructureNodeDraft)
	for _, draft := range draftMap {
		if draft.NodeNo != 1 {
			parentNodeNo := draft.ParentNodeNo
			childrenByParent[parentNodeNo] = append(childrenByParent[parentNodeNo], draft)
		}
	}

	// 按原文行号排序，保证顺序与文档一致
	for _, siblings := range childrenByParent {
		slices.SortFunc(siblings, func(a, b *vo.DocumentStructureNodeDraft) int { return a.LineNo - b.LineNo })
		// 建立前驱/后继双向指针
		for index := 0; index < len(siblings); index++ {
			current := siblings[index]
			if index != 0 {
				current.PrevSiblingNodeNo = siblings[index-1].NodeNo
			}
			if index != len(siblings)-1 {
				current.NextSiblingNodeNo = siblings[index+1].NodeNo
			}
		}
	}
}

// joinSectionPath 将父章节路径与当前章节标题以 " > " 连接，形成可读的层级路径。
func (v *TreeValidator) joinSectionPath(parentSectionPath, currentTitle string) string {
	if parentSectionPath == "" {
		return currentTitle
	}
	if currentTitle == "" {
		return parentSectionPath
	}
	return parentSectionPath + " > " + currentTitle
}

// buildPathSegment 为当前节点生成用于 URL/路径的片段（slug）。
//
// 策略优先级：
//  1. 列表类节点且有 ItemIndex → "item-N"
//  2. 拥有非空 NodeCode → slug(code)
//  3. 其他情况 → slug(displayTitle)
func (v *TreeValidator) buildPathSegment(draft *vo.DocumentStructureNodeDraft) string {
	// 列表项优先使用序号
	if draft.IsListLike() {
		if draft.ItemIndex > 0 {
			return fmt.Sprintf("item-%d", draft.ItemIndex)
		}
		return v.slug(v.displayTitle(draft))
	}

	// 存在节点代码时用代码作为路径片段
	code := strutil.Trim(draft.NodeCode)
	if code != "" {
		return v.slug(code)
	}

	// 回退：使用显示标题生成 slug
	return v.slug(v.displayTitle(draft))
}

// displayTitle 返回用于展示的标题文本。
func (v *TreeValidator) displayTitle(draft *vo.DocumentStructureNodeDraft) string {
	code := strutil.Trim(draft.NodeCode)
	title := strutil.Trim(draft.Title)

	// 标题已以前缀形式包含代码或代码为空 → 直接返回标题
	if code == "" || strings.HasPrefix(title, code) {
		return title
	}
	// 否则拼接：代码 + 空格 + 标题
	return code + " " + title
}

// slug 将任意字符串转换为 URL/路径友好的片段（只保留字母、数字、下划线、点、连字符）。
func (v *TreeValidator) slug(value string) string {
	normalized := strutil.Trim(value)
	if normalized == "" {
		return "node"
	}

	// 空白 → 连字符
	normalized = spaceToDashRegex.ReplaceAllString(normalized, "-")
	// 清除非法字符
	normalized = invalidCharRegex.ReplaceAllString(normalized, "")

	return utils.BlankToDefault(normalized, "node")
}
