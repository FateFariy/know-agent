package server

import (
	"net/http"

	"github.com/google/wire"
	"github.com/zeromicro/go-zero/rest"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/api/document"
	"github.com/swiftbit/know-agent/api/knowledge"
	"github.com/swiftbit/know-agent/internal/config"
	"github.com/swiftbit/know-agent/internal/svc"
)

var ProviderSet = wire.NewSet(NewServer, NewHTTPServer)

type Server struct {
	HTTP *rest.Server
}

func NewServer(HTTP *rest.Server) *Server {
	return &Server{HTTP: HTTP}
}

func NewHTTPServer(c *config.Config, svcCtx *svc.ServiceContext, docSrv document.HTTPServer, chatSrv chat.HTTPServer, knowledgeSrv knowledge.HTTPServer) *rest.Server {
	opts := []rest.RunOption{
		rest.WithCorsHeaders("*"),
		rest.WithCustomCors(func(header http.Header) {
			header.Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token,Authorization,Token,X-Token,X-User-Id,OS,Platform, Version")
			header.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
			header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		}, nil, "*"),
	}
	server := rest.MustNewServer(c.Http, opts...)
	document.RegisterHandlers(server, svcCtx, docSrv)
	chat.RegisterHandlers(server, svcCtx, chatSrv)
	knowledge.RegisterHandlers(server, svcCtx, knowledgeSrv)
	return server
}
