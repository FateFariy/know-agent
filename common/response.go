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

func Response(w http.ResponseWriter, resp interface{}, okMsg string, err error) {
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
