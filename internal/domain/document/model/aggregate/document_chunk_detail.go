package aggregate

import (
	"github.com/swiftbit/know-agent/common/utils"
	"github.com/swiftbit/know-agent/internal/domain/document/model/entity"
)

type DocumentChunkDetail struct {
	DocumentId    int64
	TaskId        int64
	PlanId        int64
	Chunk         *entity.DocumentChunk
	ParentChunk   *entity.DocumentParentChunk
	SiblingChunks []*entity.DocumentChunk
}

func (d *DocumentChunkDetail) FillParentInfo(parentChunk *entity.DocumentParentChunk) {
	if parentChunk != nil {
		d.ParentChunk = parentChunk
		d.ParentChunk.FillEnumName()
		d.Chunk.FillParentInfo(parentChunk)
		d.Chunk.FillEnumName()
		utils.ForEach(d.SiblingChunks, func(index int, item *entity.DocumentChunk) {
			item.FillParentInfo(parentChunk)
			item.FillEnumName()
		})
	}
}
