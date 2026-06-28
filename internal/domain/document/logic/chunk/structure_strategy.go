package chunk

import (
	"context"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

// StructureStrategy 基于文档标题结构的分块策略
// 逐行识别标题行，以标题作为天然的切分边界：
//   - 维护一个标题栈，记录当前嵌套的标题序列
//   - 遇到标题时，先输出当前累积的内容，再将新标题推入栈
//   - 生成的每个文本块都携带其所在章节路径，便于追溯
type StructureStrategy struct {
	classifier *support.DocumentLineClassifier // 行分类器，用于识别标题行
}

// NewStructureStrategy 创建结构分块策略实例
func NewStructureStrategy() *StructureStrategy {
	return &StructureStrategy{
		classifier: &support.DocumentLineClassifier{},
	}
}

// Name 返回策略名称
func (s *StructureStrategy) Name() string {
	return "STRUCTURE"
}

// Chunk 按标题结构切分文本
// 流程：
//  1. 按换行符逐行扫描 input.Text
//  2. 如果是标题行，则输出之前积累的文本，并更新当前章节路径
//  3. 否则继续累加正文到当前 chunk
//  4. 扫描结束后将最后一段文本输出
//  5. 如果未能识别出任何结构（结果为空），则降级为单个整块
func (s *StructureStrategy) Chunk(_ context.Context, input *Input, _ PipelineType) ([]*Output, error) {
	if input == nil || strings.TrimSpace(input.Text) == "" {
		return []*Output{}, nil
	}

	result := make([]*Output, 0, 8)
	headingStack := make([]string, 0, 4) // 标题栈，记录当前嵌套层级
	currentChunk := strings.Builder{}
	currentSectionPath := strings.TrimSpace(input.SectionPath)

	lines := strings.Split(input.Text, "\n")
	for _, line := range lines {
		classification := s.classifier.Classify(line)

		if classification.IsHeading() {
			// 刷出当前累积块
			s.flushChunk(&result, currentSectionPath, input.SourceType, currentChunk.String())
			currentChunk.Reset()

			// 按层级弹出同级或更高层级的标题
			classificationLevel := max(1, classification.Level)
			for len(headingStack) >= classificationLevel {
				headingStack = headingStack[:len(headingStack)-1]
			}
			headingStack = append(headingStack, classification.Title)
			currentSectionPath = composeSectionPath(input.SectionPath, strings.Join(headingStack, " > "))

			// 标题本身也加入当前块，避免空标题
			currentChunk.WriteString(strings.TrimSpace(line))
			currentChunk.WriteByte('\n')
			continue
		}

		currentChunk.WriteString(line)
		currentChunk.WriteByte('\n')
	}

	// 刷出最后一段
	s.flushChunk(&result, currentSectionPath, input.SourceType, currentChunk.String())

	if len(result) == 0 {
		// 未能识别结构，降级为单个整块
		result = append(result, &Output{
			SectionPath:   strings.TrimSpace(input.SectionPath),
			CanonicalPath: strings.TrimSpace(input.CanonicalPath),
			ItemIndex:     input.ItemIndex,
			Text:          strings.TrimSpace(input.Text),
			SourceType:    input.SourceType,
		})
	}
	return result, nil
}

// flushChunk 将累积的非空文本作为一个块加入结果
func (s *StructureStrategy) flushChunk(result *[]*Output, sectionPath string, sourceType int, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	*result = append(*result, &Output{
		SectionPath: strings.TrimSpace(sectionPath),
		ItemIndex:   0,
		Text:        trimmed,
		SourceType:  sourceType,
	})
}

// composeSectionPath 拼接基础路径与当前层级路径，用 " > " 分隔
func composeSectionPath(base, current string) string {
	baseTrimmed := strings.TrimSpace(base)
	currentTrimmed := strings.TrimSpace(current)
	if baseTrimmed == "" {
		return currentTrimmed
	}
	if currentTrimmed == "" {
		return baseTrimmed
	}
	return baseTrimmed + " > " + currentTrimmed
}
