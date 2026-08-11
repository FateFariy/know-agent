package common

import (
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type Body struct {
	Code int         `json:"code"`
	Data interface{} `json:"data,omitempty"`
	Msg  string      `json:"msg"`
}

func Response(w http.ResponseWriter, resp any, okMsg string, err error) {
	var body Body
	if err != nil {
		var bizErr *BizError
		if errors.As(err, &bizErr) {
			body.Code = bizErr.Code
			body.Msg = bizErr.Msg
		} else {
			body.Code = 1
			body.Msg = err.Error()
		}
	} else {
		body.Code = 0
		body.Data = resp
		body.Msg = okMsg
	}
	httpx.OkJson(w, body)
}

// Success 响应成功结果
func Success(w http.ResponseWriter, resp any, msg string) {
	Response(w, resp, msg, nil)
}

// Error 响应错误结果
func Error(w http.ResponseWriter, err error) {
	Response(w, nil, "", err)
}
