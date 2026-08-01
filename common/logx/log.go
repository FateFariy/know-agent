package logx

import (
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
)

func Errorf(format string, args ...any) {
	logx.Errorf(format, args...)
}

func Error(args ...any) {
	logx.Error(args...)
}

func Infof(format string, args ...any) {
	logx.Infof(format, args...)
}

func Info(args ...any) {
	logx.Info(args...)
}

func Warn(args ...any) {
	logx.Alert(fmt.Sprint(args...))
}

func Warnf(format string, args ...any) {
	logx.Alert(fmt.Sprintf(format, args...))
}
