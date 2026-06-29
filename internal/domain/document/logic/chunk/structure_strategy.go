package chunk

import (
	"context"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/document/support"
)

// StructureStrategy 基于文档标题结构的分块策略
/*
  逐行识别标题行，以标题作为天然的切分边界：
  - 维护一个标题栈，记录当前嵌套的标题序列
  - 遇到标题时，先输出当前累积的内容，再将新标题推入栈
  - 生成的每个文本块都携带其所在章节路径，便于追溯
*/
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
/*
  流程：
  1. 按换行符逐行扫描 input.Text
  2. 如果是标题行，则输出之前积累的文本，并更新当前章节路径
  3. 否则继续累加正文到当前 chunk
  4. 扫描结束后将最后一段文本输出
  5. 如果未能识别出任何结构（结果为空），则降级为单个整块
*/
func (s *StructureStrategy) Chunk(ctx context.Context, input *Input, opts ...Option) ([]*Output, error) {
	if input == nil || strutil.IsBlank(input.Text) {
		return nil, nil
	}

	result := make([]*Output, 0, 8)
	headingStack := make([]string, 0, 4) // 标题栈，记录当前嵌套层级
	currentChunk := strings.Builder{}
	currentSectionPath := strutil.Trim(input.SectionPath)

	lines := strings.Split(input.Text, "\n")
	for _, line := range lines {
		classification := s.classifier.Classify(line)

		if classification.IsHeading() {
			// 刷出当前累积块
			result = s.flushChunk(result, input.SourceType, currentSectionPath, currentChunk)

			// 按层级弹出同级或更高层级的标题
			for len(headingStack) >= classification.Level {
				headingStack = headingStack[:len(headingStack)-1]
			}
			headingStack = append(headingStack, classification.Title)
			currentSectionPath = composeSectionPath(input.SectionPath, strings.Join(headingStack, " > "))

			// 标题本身也加入当前块，避免空标题
			currentChunk.WriteString(strutil.Trim(line))
			currentChunk.WriteByte('\n')
			continue
		}

		currentChunk.WriteString(line)
		currentChunk.WriteByte('\n')
	}

	// 刷出最后一段
	result = s.flushChunk(result, input.SourceType, currentSectionPath, currentChunk)

	return result, nil
}

// flushChunk 将累积的非空文本作为一个块加入结果
func (s *StructureStrategy) flushChunk(result []*Output, sourceType int, sectionPath string, chunkBuilder strings.Builder) []*Output {
	trimmed := strutil.Trim(chunkBuilder.String())
	if trimmed == "" {
		return result
	}
	chunkBuilder.Reset()
	return append(result, &Output{
		SectionPath: sectionPath,
		Text:        trimmed,
		SourceType:  utils.Ternary(sourceType == 0, vo.ChunkSourceTypeOriginal, sourceType),
	})
}

// composeSectionPath 拼接基础路径与当前层级路径，用 " > " 分隔
func composeSectionPath(base, current string) string {
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
