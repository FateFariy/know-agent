package vo

type DocumentChunkQuery struct {
	DocumentId int64
	TaskId     int64
	PageNo     int
	PageSize   int
}
