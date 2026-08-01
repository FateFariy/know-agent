package checkpoint

import "context"

// Store 检查点存储器
type Store interface {
	// Get 获取检查点
	Get(ctx context.Context, checkPointID string) ([]byte, bool, error)

	// Set 设置检查点
	Set(ctx context.Context, checkPointID string, checkPoint []byte) error

	// Count 检查点数量
	Count(ctx context.Context, checkPointID string) (int, error)

	// Delete 删除检查点
	Delete(ctx context.Context, checkPointID string) (int, error)
}
