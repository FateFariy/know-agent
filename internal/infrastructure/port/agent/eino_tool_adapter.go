package agent

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"

	"github.com/swiftbit/know-agent/internal/domain/chat/logic/conversation"
)

// EinoToolAdapter 将通用 Tool 适配为 eino 的 tool.InvokableTool
type EinoToolAdapter[I, O any] struct {
	tool conversation.Tool[I, O]
}

// NewEinoToolAdapter 以通用工具构造 eino 适配器
func NewEinoToolAdapter[I, O any](t conversation.Tool[I, O]) *EinoToolAdapter[I, O] {
	return &EinoToolAdapter[I, O]{tool: t}
}

// Info 返回 eino 格式的工具元信息
func (t *EinoToolAdapter[I, O]) Info(ctx context.Context) (*schema.ToolInfo, error) {
	info := t.tool.Info(ctx)
	params, err := utils.GoStruct2ParamsOneOf[I]()
	if err != nil {
		return nil, err
	}
	return &schema.ToolInfo{
		Name:        info.Name,
		Desc:        info.Description,
		ParamsOneOf: params,
	}, nil
}

// InvokableRun 以 JSON 字符串调用通用工具
func (t *EinoToolAdapter[I, O]) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args I
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", err
	}
	output, err := t.tool.Invoke(ctx, args)
	invoke, err := json.Marshal(output)

	return string(invoke), err
}
