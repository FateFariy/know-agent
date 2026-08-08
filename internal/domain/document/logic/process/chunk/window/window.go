package window

import (
	"context"
	"fmt"
	"strings"

	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk"
	"github.com/swiftbit/know-agent/internal/domain/document/logic/process/chunk/recursive"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
	"github.com/swiftbit/know-agent/internal/domain/document/model/vo"
)

const Name = "WINDOW"

// Chunker 实现基于块组合+递归分割的策略
type Chunker struct {
	chunker *recursive.Chunker
	opt     *options
}

func NewChunker(chunker *recursive.Chunker) *Chunker {
	return &Chunker{
		chunker: chunker,
		opt: chunk.GetSpecificOptions(&options{
			maxChars: defaultMaxChars,
		}),
	}
}

// Name 实现 Chunker 接口
func (c *Chunker) Name() string {
	return Name
}

// Chunk 实现分块逻辑
// 解析选项
func (c *Chunker) Chunk(ctx context.Context, input *chunk.Input, opts ...chunk.Option) ([]*vo.ChunkCandidate, error) {
	if len(input.Blocks) == 0 {
		return nil, nil
	}
	opt := chunk.GetSpecificOptions(c.opt, opts...)

	var candidates []*vo.ChunkCandidate
	var window entity.DocumentBlocks
	windowChars := 0
	sep := opt.Separator

	// 刷新窗口为一个候选块
	flush := func() {
		if len(window) == 0 {
			return
		}
		var sb strings.Builder
		for i, blk := range window {
			if i > 0 {
				sb.WriteString(sep)
			}
			sb.WriteString(blk.Content)
		}

		meta := map[string]interface{}{
			"block_count": len(window),
			"nodes":       input.Nodes,
		}
		candidates = append(candidates, &chunk.ChunkCandidate{
			Content:  sb.String(),
			Metadata: meta,
		})
		window = nil
		windowChars = 0
	}

	for _, block := range input.Blocks {
		text := block.RenderBlockContent()
		if text == "" {
			continue
		}
		// 单个块超长 -> 先刷新窗口，再递归分割该块
		if len(text) > opt.maxChars {
			flush()
			splits := recursiveSplit(text, opt.MaxChars)
			for _, split := range splits {
				candidates = append(candidates, &chunk.ChunkCandidate{
					Content: split,
					Metadata: map[string]interface{}{
						"source_block_id": block.ID,
						"nodes":           input.Nodes,
					},
				})
			}
			continue
		}

		// 判断加入窗口是否超限（考虑分隔符长度）
		// 若窗口为空，直接加入；否则计算加上分隔符和当前块后的长度
		if len(window) > 0 && windowChars+len(sep)+len(text) > opt.MaxChars {
			flush()
		}
		window = append(window, block)
		// 更新窗口字符数：如果窗口之前为空，不加分隔符；否则加分隔符长度
		if len(window) == 1 {
			windowChars = len(text)
		} else {
			windowChars += len(sep) + len(text)
		}
	}
	// 最后刷新剩余窗口
	flush()

	return candidates, nil
}
