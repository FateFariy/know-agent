package vo

// RetrievalChannelResult 检索通道结果
type RetrievalChannelResult struct {
	ChannelName string           `json:"channelName"`
	Documents   []*DocumentChunk `json:"documents"`
}
