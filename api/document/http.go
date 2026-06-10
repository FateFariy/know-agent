package document

import (
	"context"
	"mime/multipart"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/swiftbit/know-agent/common"
	"github.com/swiftbit/know-agent/internal/svc"
)

type DocumentHTTPServer interface {
	// UploadDocument 上传文档
	UploadDocument(ctx context.Context, file multipart.File, header *multipart.FileHeader, req *UploadDocumentReq) (*UploadDocumentResp, error)
}

// UploadDocumentHandler 上传文档
func UploadDocumentHandler(svcCtx *svc.ServiceContext, srv DocumentHTTPServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UploadDocumentReq
		if err := httpx.Parse(r, &req); err != nil {
			common.Response(w, nil, "", err)
			return
		}

		// 最多 32 MB 不落盘，超出部分自动落盘
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			common.Response(w, nil, "", err)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			common.Response(w, nil, "", err)
			return
		}
		defer file.Close()

		resp, err := srv.UploadDocument(r.Context(), file, header, &req)
		common.Response(w, resp, "", err)
	}
}
