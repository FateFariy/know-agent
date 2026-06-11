package common

import "fmt"

var (
	ErrParm = NewBizError(10001, "参数错误：%s")
)

type BizError struct {
	Code int    // 业务错误码
	Msg  string // 业务错误描述
	Err  error  // 原始底层错误（用于错误链）
}

// NewBizError 创建业务错误
func NewBizError(code int, msg string) *BizError {
	return &BizError{Code: code, Msg: msg}
}

// NewBizErrorf 格式化创建业务错误
func NewBizErrorf(code int, format string, args ...interface{}) *BizError {
	return &BizError{
		Code: code,
		Msg:  fmt.Sprintf(format, args...),
	}
}

// Error 实现 error 接口，返回错误码和错误信息
func (e *BizError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Msg)
}

// Unwrap 获取原始底层错误
func (e *BizError) Unwrap() error {
	return e.Err
}

// Format 格式化错误信息
func (e *BizError) Format(args ...any) *BizError {
	e.Msg = fmt.Sprintf(e.Msg, args...)
	return e
}

// WrapErr 将底层原始错误，包装为带业务码的错误
func WrapErr(err error, code int, msg string) *BizError {
	if err == nil {
		return nil
	}
	return &BizError{
		Code: code,
		Msg:  msg,
		Err:  err,
	}
}
