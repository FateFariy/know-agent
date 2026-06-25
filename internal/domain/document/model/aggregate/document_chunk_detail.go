package aggregate

import (
	"github.com/duke-git/lancet/v2/slice"

	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

type DocumentChunkDetail struct {
	DocumentId    int64
	TaskId        int64
	PlanId        int64
	Chunk         *entity.DocumentChunk
	ParentBlock   *entity.DocumentParentBlock
	SiblingChunks []*entity.DocumentChunk
}

func (d *DocumentChunkDetail) FillParentInfo(parentBlock *entity.DocumentParentBlock) {
	if parentBlock != nil {
		d.ParentBlock = parentBlock
		d.ParentBlock.FillEnumName()
		d.Chunk.FillParentInfo(parentBlock)
		d.Chunk.FillEnumName()
		slice.ForEach(d.SiblingChunks, func(index int, item *entity.DocumentChunk) {
			item.FillParentInfo(parentBlock)
			item.FillEnumName()
		})
	}
}
