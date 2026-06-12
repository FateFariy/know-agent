package server

import (
	"github.com/google/wire"
	"github.com/zeromicro/go-zero/rest"

	"github.com/swiftbit/know-agent/api/document"
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

func NewHTTPServer(c config.Config, svcCtx *svc.ServiceContext, srv document.HTTPServer) *rest.Server {
	server := rest.MustNewServer(c.Http)
	document.RegisterHandlers(server, svcCtx, srv)
	return server
}
