package transform

import (
	"context"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/duke-git/lancet/v2/stream"

	"github.com/swiftbit/know-agent/common/utils"
	chatlogic "github.com/swiftbit/know-agent/internal/domain/chat/logic"
	"github.com/swiftbit/know-agent/internal/domain/chat/logic/prompt"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/svc"
)

type AmbiguityResolver struct {
	chatModel      chatlogic.ChatModelImpl[*schema.AgenticMessage]
	promptTemplate chatlogic.PromptTemplateLogic
	*ambiguityOption
}

type ambiguityOption struct {
	confidenceFloor            float64
	confidenceCeil             float64
	llmDisambiguationEnabled   bool
	maxAmbiguousSignalsPerCall int
	contextWindowLines         int
}

func WithConfidenceFloor(floor float64) TransformerOption {
	return WrapTransformerImplSpecificOptFn(func(r *ambiguityOption) {
		r.confidenceFloor = floor
	})
}

func WithConfidenceCeil(ceil float64) TransformerOption {
	return WrapTransformerImplSpecificOptFn(func(r *ambiguityOption) {
		r.confidenceCeil = ceil
	})
}

func WithLLMDisambiguationEnabled(enabled bool) TransformerOption {
	return WrapTransformerImplSpecificOptFn(func(r *ambiguityOption) {
		r.llmDisambiguationEnabled = enabled
	})
}

func WithMaxAmbiguousSignalsPerCall(max int) TransformerOption {
	return WrapTransformerImplSpecificOptFn(func(r *ambiguityOption) {
		r.maxAmbiguousSignalsPerCall = max
	})
}

func WithContextWindowLines(lines int) TransformerOption {
	return WrapTransformerImplSpecificOptFn(func(r *ambiguityOption) {
		r.contextWindowLines = lines
	})
}

func NewAmbiguityResolver(svcCtx *svc.ServiceContext, chatModel chatlogic.ChatModelImpl[*schema.AgenticMessage], promptTemplate chatlogic.PromptTemplateLogic) *AmbiguityResolver {
	return &AmbiguityResolver{
		chatModel:      chatModel,
		promptTemplate: promptTemplate,
		ambiguityOption: &ambiguityOption{
			confidenceFloor:            svcCtx.Config.StructureParsing.AmbiguityConfidenceFloor,
			confidenceCeil:             svcCtx.Config.StructureParsing.AmbiguityConfidenceCeil,
			llmDisambiguationEnabled:   svcCtx.Config.StructureParsing.LLMDisambiguationEnabled,
			maxAmbiguousSignalsPerCall: svcCtx.Config.StructureParsing.MaxAmbiguousSignalsPerCall,
			contextWindowLines:         svcCtx.Config.StructureParsing.ContextWindowLines,
		},
	}
}

// Transform 对信号进行 LLM 二义性消解
/* 整体流程：
   1. 若没有信号或未启用 LLM 消解 → 直接原路返回
   2. 筛选「处于目标置信度区间 + 被标记为 ambiguous」的信号，且单次最多处理 maxAmbiguousSignalsPerCall 条
   3. 为每个候选信号构建上下文窗口（前后若干行），渲染为提示语所需的 candidateBlocks
   4. 组装用户提示并调用 LLM 生成结构化结果
   5. 解析 LLM 返回的 JSON 数组，按 LineNo 建立行号→结果的映射
   6. 遍历原信号，将命中的消解结果回填（kind、levelHint、置信度等）
   返回：处理后的信号切片（顺序与 sourceSignals 一致）+ 错误
*/
func (r *AmbiguityResolver) Transform(ctx context.Context, documentTitle string, allLines []string,
	sourceSignals []*vo.DocumentStructureSignal, opts ...TransformerOption) ([]*vo.DocumentStructureSignal, error) {
	if len(sourceSignals) == 0 {
		return sourceSignals, nil
	}

	// 聚合可选参数（允许调用方覆盖默认值）
	opt := GetTransformerImplSpecificOptions[ambiguityOption](r.ambiguityOption, opts...)

	// LLM 消解未开启 → 直接透传
	if !opt.llmDisambiguationEnabled {
		return sourceSignals, nil
	}

	// 筛选 ambiguous 且置信度在目标区间内的信号，并控制单次请求上限（控制 token 与耗时）
	ambiguousSignals := stream.FromSlice(sourceSignals).
		Filter(func(signal *vo.DocumentStructureSignal) bool {
			return signal.IsAmbiguous() && signal.Confidence >= opt.confidenceFloor && signal.Confidence <= opt.confidenceCeil
		}).Limit(opt.maxAmbiguousSignalsPerCall).ToSlice()

	if len(ambiguousSignals) == 0 {
		return sourceSignals, nil
	}

	// 构建候选块文本（含行号标记 >> ），用于提示语输入
	candidateBlocks, err := r.buildCandidateBlocks(ambiguousSignals, allLines)
	if err != nil {
		return nil, err
	}

	// 渲染用户提示：注入文档标题与候选块
	userPrompt, err := r.promptTemplate.Render(prompt.DocumentStructureAmbiguity, map[string]any{
		"documentTitle":   utils.BlankToDefault(documentTitle, "未命名文档"),
		"candidateBlocks": candidateBlocks,
	})
	if err != nil {
		return nil, err
	}

	// 调用 LLM 获取结构化判定结果
	content, err := r.chatModel.Generate(ctx, "", userPrompt)
	if err != nil {
		return nil, err
	}

	// 从 LLM 原始输出中解析结果
	var results []*vo.DisambiguationResult
	if err = utils.Unmarshal(content, &results); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return sourceSignals, nil
	}

	// 按 LineNo 构建映射
	resultMap := utils.SliceToMapBy(results, func(result *vo.DisambiguationResult) (int, *vo.DisambiguationResult) {
		return result.LineNo, result
	})

	// 遍历原信号切片，将消解结果应用到对应的信号上（保持原顺序）
	merged := make([]*vo.DocumentStructureSignal, len(sourceSignals))
	for i, signal := range sourceSignals {
		if signal != nil {
			merged[i] = r.applyResult(signal, resultMap[signal.LineNo])
		}
	}

	return merged, nil
}

// buildCandidateBlocks 为每个 ambiguous 信号构造上下文窗口候选文本块
/*
  构造策略：
  - 以 signal.LineNo 作为目标行，向前/向后各取 contextWindowLines 行
  - 目标行前缀使用 >> 标记，其他行使用 3 个空格，便于 LLM 定位
  - 每个信号渲染为一个候选块（包含初始 kind、title、code 等）
  返回：拼接好的完整候选块文本
*/
func (r *AmbiguityResolver) buildCandidateBlocks(ambiguousSignals []*vo.DocumentStructureSignal, allLines []string) (string, error) {
	var sb strings.Builder

	contextWindow := max(r.contextWindowLines, 1)
	// 遍历每个 ambiguous 信号，生成对应的上下文窗口块
	for _, signal := range ambiguousSignals {
		if signal == nil {
			continue
		}

		// 将行号转换为索引（行号从 1 开始，索引从 0 开始）
		currentIndex := max(signal.LineNo-1, 0)
		start := max(currentIndex-contextWindow, 0)
		end := min(currentIndex+contextWindow, len(allLines)-1)

		// 逐行拼接上下文窗口，目标行使用 >> 前缀高亮
		var contextBuilder strings.Builder
		for index := start; index <= end; index++ {
			prefix := utils.Ternary(index+1 == signal.LineNo, ">> ", "   ")
			contextBuilder.WriteString(prefix)
			contextBuilder.WriteString(strconv.Itoa(index + 1))
			contextBuilder.WriteString(": ")
			contextBuilder.WriteString(allLines[index])
			contextBuilder.WriteString("\n")
		}

		// 将窗口上下文与信号元信息渲染为候选块模板
		render, err := r.promptTemplate.Render(prompt.DocumentStructureAmbiguityCandidate, map[string]any{
			"lineNo":       signal.LineNo,
			"contextLines": strings.TrimSpace(contextBuilder.String()),
			"initialKind":  vo.SignalKindName(signal.Kind),
			"initialTitle": signal.Title,
			"initialCode":  signal.NodeCode,
		})
		if err != nil {
			return "", err
		}
		sb.WriteString(render)
		sb.WriteString("\n\n")
	}

	return strings.TrimSpace(sb.String()), nil
}

// applyResult 将 LLM 消解结果应用到源信号
func (r *AmbiguityResolver) applyResult(source *vo.DocumentStructureSignal, resolved *vo.DisambiguationResult) *vo.DocumentStructureSignal {
	if source == nil || resolved == nil || resolved.ResolvedKind == "" {
		return source
	}

	// 取值归一化（大小写不敏感）
	source.Kind = resolved.ToSignalKind()

	// 仅在判定为标题时覆盖，避免误改列表项或正文的层级
	if source.Kind == vo.SignalKindHeading && resolved.LevelHint > 0 {
		source.LevelHint = resolved.LevelHint
	}

	// 记录来源原因，并将置信度拉到「消解后可信」的最低水位
	source.Reasons = append(source.Reasons, "llm-disambiguated")
	source.Confidence = max(source.Confidence, 0.88)

	return source
}
