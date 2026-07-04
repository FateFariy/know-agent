package conversation

import (
	"fmt"

	"github.com/swiftbit/know-agent/internal/domain/chat/model/vo"
)

// ExecutorRegistry 对话执行器注册表。
// 按执行模式查找对应执行器，确保上层能统一入口完成不同策略的问答。
type ExecutorRegistry struct {
	executors map[vo.ExecutionMode]Executor
}

func NewExecutorRegistry(executors ...Executor) *ExecutorRegistry {
	reg := &ExecutorRegistry{
		executors: make(map[vo.ExecutionMode]Executor, len(executors)),
	}
	for _, exec := range executors {
		if exec != nil {
			reg.executors[exec.Mode()] = exec
		}
	}
	return reg
}

// Register 动态追加执行器（供未来扩展使用）
func (r *ExecutorRegistry) Register(exec Executor) {
	if exec == nil || r.executors == nil {
		return
	}
	r.executors[exec.Mode()] = exec
}

// Get 根据模式获取执行器，找不到返回错误
func (r *ExecutorRegistry) Get(mode vo.ExecutionMode) (Executor, error) {
	if r.executors == nil {
		return nil, fmt.Errorf("未找到执行模式对应的执行器: %s", mode.String())
	}
	exec, ok := r.executors[mode]
	if !ok || exec == nil {
		return nil, fmt.Errorf("未找到执行模式对应的执行器: %s", mode.String())
	}
	return exec, nil
}
