package channel

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	cvo "github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
	"github.com/swiftbit/know-agent/internal/domain/rag/model/vo"
)

// RetrievalChannel 检索通道接口
type RetrievalChannel interface {
	// ChannelName 检索通道名称
	ChannelName() string

	// Supports 是否支持该执行计划
	Supports(plan *cvo.ConversationExecutionPlan) bool

	// Retrieve 根据子问题检索
	Retrieve(ctx context.Context, query *vo.DocumentRetrieve) (*vo.RetrievalChannelResult, error)
}

func Warnf(format string, v ...any) {
	logx.Alert(fmt.Sprintf(format, v...))
}
