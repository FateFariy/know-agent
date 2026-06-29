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
func (s *Strategy) Chunk(ctx context.Context, input *chunk.Input, opts ...chunk.Option) ([]*chunk.Output, error) {
	if input == nil || strutil.IsBlank(input.Text) {
		return nil, nil
	}

	result := make([]*chunk.Output, 0, 8)
	headingStack := make([]string, 0, 4)
	currentSectionPath := strutil.Trim(input.SectionPath)
	currentChunk := make([]rune, 0, 512)

	lines := strings.Split(input.Text, "\n")
	for _, line := range lines {
		classification := s.classifier.Classify(line)

		if classification.IsHeading() {
			// 刷出当前累积块
			result = s.flushChunk(result, input.SourceType, currentSectionPath, currentChunk)
			currentChunk = currentChunk[:0]

			// 按层级弹出同级或更高层级的标题
			for len(headingStack) >= classification.Level {
				headingStack = headingStack[:len(headingStack)-1]
			}
			headingStack = append(headingStack, classification.Title)
			currentSectionPath = s.composeSectionPath(input.SectionPath, strings.Join(headingStack, " > "))

			// 标题本身也加入当前块，避免空标题
			currentChunk = append(currentChunk, []rune(strutil.Trim(line))...)
			currentChunk = append(currentChunk, '\n')
			continue
		}

		currentChunk = append(currentChunk, []rune(line)...)
		currentChunk = append(currentChunk, '\n')
	}

	// 刷出最后一段
	result = s.flushChunk(result, input.SourceType, currentSectionPath, currentChunk)
	if len(result) == 0 {
		// 未能识别结构，降级为单个整块
		result = append(result, &chunk.Output{
			SectionPath:   strutil.Trim(input.SectionPath),
			CanonicalPath: strutil.Trim(input.CanonicalPath),
			ItemIndex:     input.ItemIndex,
			Text:          strutil.Trim(input.Text),
			SourceType:    input.SourceType,
		})
	}
	return result, nil
}

// flushChunk 将累积的非空文本作为一个块加入结果
func (s *Strategy) flushChunk(result []*chunk.Output, sourceType int, sectionPath string, currentChunk []rune) []*chunk.Output {
	trimmed := strutil.Trim(string(currentChunk))
	if trimmed == "" {
		return result
	}
	return append(result, &chunk.Output{
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
