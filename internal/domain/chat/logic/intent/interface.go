package intent

import (
	"context"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// Recognizer 意图识别策略接口
type Recognizer interface {
	// Name 返回提供者名称，用于日志和溯源
	Name() string

	// Recognize 根据输入的意图识别输入，返回意图识别结果
	Recognize(ctx context.Context, input *RecognitionInput) (*vo.IntentRecognitionResult, error)
}
