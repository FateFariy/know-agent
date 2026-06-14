package support

import "fmt"

// SafeEmitNext 安全地向 channel 发送数据
func SafeEmitNext(stream chan<- string, payload string) error {
	if stream == nil || payload == "" {
		return nil
	}

	// 非阻塞发送：如果 channel 已满或没有接收者，会阻塞
	select {
	case stream <- payload:
		// 发送成功
	default:
		// 当前无法立即发送（channel 满，无接收者）
		return fmt.Errorf("流式事件发送失败: channel 可能无接收者或缓冲区满")
	}
	return nil
}

// SafeEmitComplete 安全地关闭 channel
func SafeEmitComplete(stream chan<- string) {
	if stream == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			return
		}
	}()
	close(stream)
}
