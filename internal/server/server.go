package server

import (
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

func NewHTTPServer(c config.Config, svcCtx *svc.ServiceContext, docSrv document.HTTPServer, chatSrv chat.HTTPServer, knowledgeSrv knowledge.HTTPServer) *rest.Server {
	server := rest.MustNewServer(c.Http)
	document.RegisterHandlers(server, svcCtx, docSrv)
	chat.RegisterHandlers(server, svcCtx, chatSrv)
	knowledge.RegisterHandlers(server, svcCtx, knowledgeSrv)
	return server
}
