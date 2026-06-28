package transform

import (
	"context"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

// DocumentStructureNodeExtractor 文档结构节点抽取器：编排信号抽取、歧义消解、层级构建与树验证， 最终输出可用于下游渲染/检索的候选节点列表
/*
  职责定位：
  - 作为 document/structure 子系统的门面（Facade），对外暴露统一的 Extract 入口
  - 组合四个核心组件：SignalExtractor / AmbiguityResolver / HierarchyResolver / TreeValidator
  - 处理短文本（空内容）的退化场景：仅返回文档根节点，避免无意义的计算开销
*/
type DocumentStructureNodeExtractor struct {
	signalExtractor   *SignalExtractor   // 信号抽取：将原始文本拆分为标题/列表/正文等结构信号
	ambiguityResolver *AmbiguityResolver // 歧义消解：对候选标题进行 LLM 二次判定（若配置启用）
	hierarchyResolver *HierarchyResolver // 层级构建：基于信号流组装父子关系与嵌套列表
	treeValidator     *TreeValidator     // 树验证：规范化父子关系、深度、路径与兄弟链表
}

// NewDocumentStructureNodeExtractor 构造 DocumentStructureNodeExtractor
func NewDocumentStructureNodeExtractor(signalExtractor *SignalExtractor, ambiguityResolver *AmbiguityResolver,
	hierarchyResolver *HierarchyResolver, treeValidator *TreeValidator) *DocumentStructureNodeExtractor {
	return &DocumentStructureNodeExtractor{
		signalExtractor:   signalExtractor,
		ambiguityResolver: ambiguityResolver,
		hierarchyResolver: hierarchyResolver,
		treeValidator:     treeValidator,
	}
}

// Extract 执行文档结构节点抽取：输入文档标题与原始文本，输出结构候选节点列表。
/*
  整体流程：
  1. 规范化标题与正文文本，提供稳定的后续处理输入
  2. 短文本退化处理：当正文为空时，直接返回单一文档根节点（避免后续组件做无意义计算）
  3. 信号抽取：SignalExtractor 将文本切分为逻辑行并识别结构信号（标题/列表/正文等）
  4. 歧义消解：AmbiguityResolver 对不确定的候选标题做 LLM 判定
  5. 层级构建：HierarchyResolver 将扁平信号流组织成带父子关系的节点草稿
  6. 树验证：TreeValidator 规范化 Draft 树并输出最终的候选节点列表

  返回：按节点编号升序排列的候选节点列表（含文档根节点）
*/
func (e *DocumentStructureNodeExtractor) Extract(ctx context.Context, documentTitle, parsedText string, opts ...TransformerOption) []*vo.DocumentStructureNodeCandidate {
	normalizedTitle := strutil.Trim(documentTitle)
	if normalizedTitle == "" {
		normalizedTitle = "文档"
	}
	normalizedText := strutil.Trim(parsedText)

	// 短文本退化：无正文内容时直接返回文档根节点（避免噪声处理）
	if normalizedText == "" {
		return []*vo.DocumentStructureNodeCandidate{
			{
				NodeNo:        1,
				NodeType:      vo.NodeTypeDocument,
				Title:         normalizedTitle,
				AnchorText:    normalizedTitle,
				CanonicalPath: "/document",
			},
		}
	}

	// 信号抽取 — 获得结构信号批量（含原始上下文行）
	signalBatch := e.signalExtractor.Transform(ctx, parsedText, opts...)

	// 歧义消解 — 对信号中的候选项做 LLM 二次判定（若配置/实例可用）
	resolvedSignals, _ := e.ambiguityResolver.Transform(ctx, documentTitle, signalBatch.ContextLines, signalBatch.Signals, opts...)
	if resolvedSignals == nil {
		resolvedSignals = signalBatch.Signals
	}

	// 层级构建 — 将信号流转为草稿节点树（含根节点）
	drafts := e.hierarchyResolver.Transform(normalizedTitle, resolvedSignals, opts...)

	// 树验证与规范化 — 生成最终候选节点列表
	return e.treeValidator.Transform(normalizedTitle, drafts, opts...)
}
