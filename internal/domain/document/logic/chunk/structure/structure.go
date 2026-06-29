package structure

import (
	"context"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/chunk"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

const (
	Name = "STRUCTURE"
)

// Strategy 基于文档标题结构的分块策略。
/*
  逐行识别标题行，以标题作为天然的切分边界：
  - 维护一个标题栈，记录当前嵌套的标题序列
  - 遇到标题时，先输出当前累积的内容，再将新标题推入栈
  - 生成的每个文本块都携带其所在章节路径，便于追溯
*/
type Strategy struct {
	classifier *support.DocumentLineClassifier
}

// NewStrategy 创建结构分块策略实例
func NewStrategy(opts ...chunk.Option) *Strategy {
	return &Strategy{
		classifier: &support.DocumentLineClassifier{},
	}
}

// Name 返回策略名称
func (s *Strategy) Name() string {
	return Name
}

// Chunk 按标题结构切分文本
func (s *Strategy) Chunk(ctx context.Context, input *chunk.TextBlock, opts ...chunk.Option) ([]*chunk.TextBlock, error) {
	if input == nil || strutil.IsBlank(input.Text) {
		return nil, nil
	}

	result := make([]*chunk.TextBlock, 0, 8)
	headingStack := make([]string, 0, 4)
	currentSectionPath := strutil.Trim(input.SectionPath)
	currentChunk := strings.Builder{}

	lines := strings.Split(input.Text, "\n")
	for _, line := range lines {
		classification := s.classifier.Classify(line)

		if classification.IsHeading() {
			// 刷出当前累积块
			result = s.flushChunk(result, input.SourceType, currentSectionPath, currentChunk)
			currentChunk.Reset()

			// 按层级弹出同级或更高层级的标题
			for len(headingStack) >= classification.Level {
				headingStack = headingStack[:len(headingStack)-1]
			}
			headingStack = append(headingStack, classification.Title)
			currentSectionPath = s.composeSectionPath(input.SectionPath, strings.Join(headingStack, " > "))

			// 标题本身也加入当前块，避免空标题
			currentChunk.WriteString(strutil.Trim(line))
			currentChunk.WriteRune('\n')
			continue
		}

		currentChunk.WriteString(line)
		currentChunk.WriteRune('\n')
	}

	// 刷出最后一段
	result = s.flushChunk(result, input.SourceType, currentSectionPath, currentChunk)

	return result, nil
}

// flushChunk 将累积的非空文本作为一个块加入结果
func (s *Strategy) flushChunk(result []*chunk.TextBlock, sourceType int, sectionPath string, currentChunk strings.Builder) []*chunk.TextBlock {
	trimmed := strutil.Trim(currentChunk.String())
	if trimmed == "" {
		return result
	}
	currentChunk.Reset()
	return append(result, &chunk.TextBlock{
		SectionPath: sectionPath,
		Text:        trimmed,
		SourceType:  utils.Ternary(sourceType == 0, vo.ChunkSourceTypeOriginal, sourceType),
	})
}

// composeSectionPath 拼接基础路径与当前层级路径，用 " > " 分隔
func (s *Strategy) composeSectionPath(base, current string) string {
	baseTrimmed := strutil.Trim(base)
	currentTrimmed := strutil.Trim(current)
	if baseTrimmed == "" {
		return currentTrimmed
	}
	if currentTrimmed == "" {
		return baseTrimmed
	}
	return baseTrimmed + " > " + currentTrimmed
}
