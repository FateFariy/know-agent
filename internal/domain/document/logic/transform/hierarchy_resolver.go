package transform

import (
	"slices"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

type listContext struct {
	node        *vo.DocumentStructureNodeDraft
	indentLevel int
}

type HierarchyResolver struct{}

func NewHierarchyResolver() *HierarchyResolver {
	return &HierarchyResolver{}
}

// Transform 将扁平信号流构造成具有层级关系的节点草稿（根节点 + 章节 + 列表项）
/*
  处理顺序：按 Kind 分派 → Blank 清理列表状态 → Noise 丢弃 → 表格/引用/正文挂载正文 → 列表项入栈 → 标题重置当前章节并清理列表
  返回：按出现顺序组织的节点草稿（含根节点），最终会按 LineNo 排序
*/
func (r *HierarchyResolver) Transform(documentTitle string, signals []*vo.DocumentStructureSignal, opts ...TransformerOption) []*vo.DocumentStructureNodeDraft {
	drafts := make([]*vo.DocumentStructureNodeDraft, 0)

	// 构造文档根节点（NodeNo=1，后续父子关系都以此为最低公共祖先）
	root := &vo.DocumentStructureNodeDraft{
		NodeNo:        1,
		NodeType:      vo.NodeTypeDocument,
		Title:         utils.BlankToDefault(documentTitle, "文档"),
		AnchorText:    utils.BlankToDefault(documentTitle, "文档"),
		CanonicalPath: "/document",
		SourceFamily:  "document",
		Confidence:    1.0,
	}
	drafts = append(drafts, root)

	// 初始化遍历状态
	nextNodeNo := 2                                    // 下一个可用的节点编号（根为 1，所以从 2 开始）
	currentSection := root                             // 当前激活的章节/章节候选，正文/列表默认挂到它下面
	var currentListItem *vo.DocumentStructureNodeDraft // 当前最近的列表项，用于挂接列表续行
	listStack := make([]*listContext, 0)               // 列表上下文栈（按缩进级别维护），缩进减少时回溯到更外层父节点
	latestHeadingByDepth := make(map[int]int)          // 按层级深度记录最近的标题节点编号，用于按深度回找父章节
	latestHeadingByNumericPath := make(map[string]int) // 按数字路径（如 1.2.3）记录最近的标题节点编号，用于数字型父子关系

	// 逐行扫描信号：按 Kind 分发处理
	for _, signal := range signals {
		// 跳过空信号与未定位到行的信号
		if signal == nil || signal.LineNo == 0 {
			continue
		}

		switch signal.Kind {
		// 空行：视为列表结束，清空列表项指针与缩进栈，避免跨段落的错误嵌套
		case vo.SignalKindBlank:
			currentListItem = nil
			listStack = listStack[:0]

		// 噪声：直接忽略
		case vo.SignalKindNoise:
			continue

		// 表格/引用/正文：统一挂载为正文行
		case vo.SignalKindTableRow, vo.SignalKindQuote, vo.SignalKindBody:
			r.appendBody(signal, currentSection, currentListItem, root)

		// 步骤项 / 列表项：先按缩进栈确定父节点，再创建并登记列表节点
		case vo.SignalKindStepItem, vo.SignalKindListItem:
			listParent := r.resolveListParent(signal, listStack, currentSection, root)
			listNode := r.buildListNode(signal, nextNodeNo, listParent)
			nextNodeNo++
			drafts = append(drafts, listNode)
			// 更新当前列表项为新节点，并维护缩进栈
			currentListItem = listNode
			listStack = r.registerListContext(signal, listNode, listStack)
			// 章节的 Lines 同步追加，保证章节级文本聚合完整性
			if currentSection != nil {
				currentSection.AppendLine(signal.NormalizedText)
			}

		// 标题 / 候选标题：计算深度与父节点后创建章节节点，并重置列表状态
		case vo.SignalKindHeading, vo.SignalKindHeadingCandidate:
			headingNode := r.buildHeadingNode(signal, nextNodeNo, drafts, latestHeadingByDepth, latestHeadingByNumericPath)
			nextNodeNo++
			drafts = append(drafts, headingNode)
			// 切换当前章节至新标题；列表状态清空（标题与列表不在同一上下文）
			currentSection = headingNode
			currentListItem = nil
			listStack = listStack[:0]

		// 未知 Kind 兜底：当作正文挂载，避免信号丢失
		default:
			r.appendBody(signal, currentSection, currentListItem, root)
		}
	}

	// 按 LineNo 排序，确保最终按文档顺序呈现（处理过程中父子编号可能跨段落插入）
	slices.SortFunc(drafts, func(a, b *vo.DocumentStructureNodeDraft) int { return a.LineNo - b.LineNo })

	return drafts
}

// appendBody 将正文行挂到合适的节点上，优先级：当前列表项 > 当前章节 > 根节点
func (r *HierarchyResolver) appendBody(signal *vo.DocumentStructureSignal, currentSection, currentListItem, root *vo.DocumentStructureNodeDraft) {
	line := signal.NormalizedText
	if line == "" {
		return
	}

	// 选定主挂载目标：列表项 → 章节 → 根
	var target *vo.DocumentStructureNodeDraft
	if currentListItem != nil {
		target = currentListItem
	} else if currentSection != nil {
		target = currentSection
	} else {
		target = root
	}
	target.AppendLine(line)

	// 如果当前挂到了列表项且存在章节 → 章节也同步一份（保证章节聚合完整）
	if currentListItem != nil && currentSection != nil && currentSection.NodeNo != currentListItem.NodeNo {
		currentSection.AppendLine(line)
	}

	// 当前章节不存在但目标不是 root（例如还没有遇到标题前的散行）→ 额外写入根节点，避免丢失
	if currentSection == nil && target != root {
		root.AppendLine(line)
	}
}

// resolveListParent 根据缩进栈确定当前列表项的父节点
func (r *HierarchyResolver) resolveListParent(signal *vo.DocumentStructureSignal, listStack []*listContext,
	currentSection, root *vo.DocumentStructureNodeDraft) *vo.DocumentStructureNodeDraft {
	indentLevel := utils.Ternary(signal.IndentLevel < 0, 0, signal.IndentLevel)
	// 弹出所有缩进 ≥ 当前的条目，保证「同级或更外层」
	for len(listStack) > 0 && listStack[len(listStack)-1].indentLevel >= indentLevel {
		listStack = listStack[:len(listStack)-1]
	}

	// 栈中存在更小缩进的条目 → 使用其作为父节点（缩进更深，形成嵌套）
	if len(listStack) > 0 && indentLevel > listStack[len(listStack)-1].indentLevel {
		return listStack[len(listStack)-1].node
	}

	// 栈空 → 父为章或根
	return utils.Ternary(currentSection != nil, currentSection, root)
}

// buildListNode 根据信号创建列表项/步骤项节点草稿
func (r *HierarchyResolver) buildListNode(signal *vo.DocumentStructureSignal, nodeNo int, parent *vo.DocumentStructureNodeDraft) *vo.DocumentStructureNodeDraft {
	draft := &vo.DocumentStructureNodeDraft{
		NodeNo:       nodeNo,
		LineNo:       signal.LineNo,
		NodeType:     utils.Ternary(signal.Kind == vo.SignalKindStepItem, vo.NodeTypeStep, vo.NodeTypeListItem),
		ParentNodeNo: utils.Ternary(parent != nil, parent.NodeNo, 1),
		Depth:        utils.Ternary(parent != nil, parent.Depth+1, 1),
		NodeCode:     utils.BlankToDefault(signal.NodeCode, utils.Ternary(signal.ItemIndex == 0, "", strconv.Itoa(signal.ItemIndex))),
		Title:        signal.Title,
		AnchorText:   utils.BlankToDefault(signal.NormalizedText, signal.Title),
		ItemIndex:    signal.ItemIndex,
		SourceFamily: utils.Ternary(signal.Kind == vo.SignalKindStepItem, "step", "list"),
		Confidence:   signal.Confidence,
	}
	draft.AppendLine(signal.NormalizedText)
	return draft
}

// registerListContext 在列表栈中登记当前节点，用于后续项判定父节点；返回更新后的栈
// 栈维护策略：先弹出缩进 ≥ 自身的条目，再将自己压栈
func (r *HierarchyResolver) registerListContext(signal *vo.DocumentStructureSignal, listNode *vo.DocumentStructureNodeDraft, listStack []*listContext) []*listContext {
	indentLevel := utils.Ternary(signal.IndentLevel < 0, 0, signal.IndentLevel)
	for len(listStack) > 0 && listStack[len(listStack)-1].indentLevel >= indentLevel {
		listStack = listStack[:len(listStack)-1]
	}
	return append(listStack, &listContext{
		node:        listNode,
		indentLevel: indentLevel,
	})
}

// buildHeadingNode 构造章节节点草稿并更新标题上下文表
/*
  步骤：
  1. 根据 family / 数字路径 / 上下文确定标题深度 depth
  2. 根据同样策略确定父节点编号 parentNodeNo
  3. 组装 Draft 并写入首行文本
  4. 维护 latestHeadingByDepth：清理所有 >= 当前 depth 的条目，再登记当前 nodeNo
  5. 维护 latestHeadingByNumericPath：若有 numericPath 则登记对应的节点编号
*/
func (r *HierarchyResolver) buildHeadingNode(signal *vo.DocumentStructureSignal, nodeNo int, drafts []*vo.DocumentStructureNodeDraft,
	latestHeadingByDepth map[int]int, latestHeadingByNumericPath map[string]int) *vo.DocumentStructureNodeDraft {
	// 计算深度（由 family + numericPath 决定）
	depth := r.resolveHeadingDepth(signal, drafts, latestHeadingByNumericPath)
	// 计算父节点编号（对称策略）
	parentNodeNo := r.resolveHeadingParentNodeNo(signal, depth, latestHeadingByDepth, latestHeadingByNumericPath)
	// 组装章节节点草稿
	draft := &vo.DocumentStructureNodeDraft{
		NodeNo:       nodeNo,
		LineNo:       signal.LineNo,
		NodeType:     vo.NodeTypeSection,
		ParentNodeNo: parentNodeNo,
		Depth:        depth,
		NodeCode:     signal.NodeCode,
		Title:        signal.Title,
		AnchorText:   r.buildHeadingAnchorText(signal),
		NumericPath:  slices.Clone(signal.NumericPath),
		SourceFamily: r.resolveHeadingFamily(signal),
		Confidence:   signal.Confidence,
	}
	draft.AppendLine(signal.NormalizedText)
	// 清理同深度及更深的旧标题（新标题已出现，旧层级无效）
	for k := range latestHeadingByDepth {
		if k >= depth {
			delete(latestHeadingByDepth, k)
		}
	}
	latestHeadingByDepth[depth] = nodeNo
	// 如果存在数字路径，登记到 numericPath → nodeNo 映射
	numericKey := support.NumericKey(signal.NumericPath)
	if numericKey != "" {
		latestHeadingByNumericPath[numericKey] = nodeNo
	}
	return draft
}

// resolveHeadingDepth 解析标题的层级深度
/*
  策略：
  - markdown 家族：直接基于 LevelHint，最小为 1
  - chapter / appendix：始终为 1（根级章节）
  - decimal（数字编号型）：优先回查父节点（按上一级数字路径或首段章节）深度+1，
    回查失败则退化为 len(numericPath)
  - 其他：退化为 LevelHint 约束（最小为 1）
*/
func (r *HierarchyResolver) resolveHeadingDepth(signal *vo.DocumentStructureSignal, drafts []*vo.DocumentStructureNodeDraft, latestHeadingByNumericPath map[string]int) int {
	family := r.resolveHeadingFamily(signal)
	numericPath := signal.NumericPath

	// Markdown 标题：使用 LevelHint（# 的数量），保证 ≥1
	if family == "markdown" {
		return max(1, signal.LevelHint)
	}

	// 章/附录：固定为根级章节
	if family == "chapter" || family == "appendix" {
		return 1
	}

	// 数字编号型（例如 1.2.3）：多级深度由父节点推导
	if family == "decimal" {
		// 只有一级 → 等同于章级
		if len(numericPath) <= 1 {
			return 1
		}

		// 尝试「上一级数字路径」精确匹配父节点
		parentKey := support.NumericKey(numericPath[:len(numericPath)-1])
		if parentNodeNo, ok := latestHeadingByNumericPath[parentKey]; ok {
			if parent, exists := r.findByNodeNo(drafts, parentNodeNo); exists {
				return parent.Depth + 1
			}
		}

		// 退回到章级（第 1 段的数字路径：例如 1）
		chapterKey := strconv.Itoa(numericPath[0])
		if chapterParent, ok := latestHeadingByNumericPath[chapterKey]; ok {
			if parent, exists := r.findByNodeNo(drafts, chapterParent); exists {
				return parent.Depth + 1
			}
		}
		// 所有回查失败，使用数字路径长度作为启发式深度
		return len(numericPath)
	}

	// plain / 其他：回退到 LevelHint
	return max(1, signal.LevelHint)
}

// resolveHeadingParentNodeNo 解析标题的父节点编号
/*
  策略：
  - chapter / appendix：固定父为根（NodeNo=1）
  - decimal 多级：优先按上一级数字路径精确匹配，其次按首段章节匹配
  - 其他：从 depth-1 向下回查 latestHeadingByDepth，找到最近的上层标题；若无，父为根
*/
func (r *HierarchyResolver) resolveHeadingParentNodeNo(signal *vo.DocumentStructureSignal, depth int,
	latestHeadingByDepth map[int]int, latestHeadingByNumericPath map[string]int) int {
	family := r.resolveHeadingFamily(signal)
	numericPath := signal.NumericPath

	// 章/附录：直接以文档根为父
	if family == "chapter" || family == "appendix" {
		return 1
	}

	// 数字编号型多级标题：尝试按数字路径精确回找父
	if family == "decimal" && len(numericPath) > 1 {
		exactParentKey := support.NumericKey(numericPath[:len(numericPath)-1])
		if exactParent, ok := latestHeadingByNumericPath[exactParentKey]; ok {
			return exactParent
		}

		// 找不到上一级时，退到首段章节
		chapterParentKey := strconv.Itoa(numericPath[0])
		if chapterParent, ok := latestHeadingByNumericPath[chapterParentKey]; ok {
			return chapterParent
		}
	}

	// 默认策略：按深度降序回查最近的上层标题（Markdown 与 plain）
	for candidateDepth := depth - 1; candidateDepth > 0; candidateDepth-- {
		if parentNodeNo, ok := latestHeadingByDepth[candidateDepth]; ok {
			return parentNodeNo
		}
	}

	// 回退到文档根
	return 1
}

// resolveHeadingFamily 根据信号的 Reasons 推断标题家族
func (r *HierarchyResolver) resolveHeadingFamily(signal *vo.DocumentStructureSignal) string {
	for _, reason := range signal.Reasons {
		switch reason {
		case "markdown-heading":
			return "markdown"
		case "chapter-heading":
			return "chapter"
		case "appendix-heading":
			return "appendix"
		case "decimal-heading":
			return "decimal"
		case "single-digit-ambiguous-heading":
			return "decimal"
		}
	}
	return "plain"
}

// buildHeadingAnchorText 生成章节的锚文本
func (r *HierarchyResolver) buildHeadingAnchorText(signal *vo.DocumentStructureSignal) string {
	code := strutil.Trim(signal.NodeCode)
	title := strutil.Trim(signal.Title)

	if code == "" || strings.HasPrefix(title, code) {
		return title
	}
	return code + " " + title
}

// findByNodeNo 在 drafts 中按 NodeNo 线性查找
func (r *HierarchyResolver) findByNodeNo(drafts []*vo.DocumentStructureNodeDraft, nodeNo int) (*vo.DocumentStructureNodeDraft, bool) {
	return slice.FindBy(drafts, func(index int, draft *vo.DocumentStructureNodeDraft) bool {
		return draft != nil && draft.NodeNo == nodeNo
	})
}
