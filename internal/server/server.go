package server

import (
	"net/http"

	"github.com/google/wire"
	"github.com/zeromicro/go-zero/rest"

	"github.com/swiftbit/know-agent/api/chat"
	"github.com/swiftbit/know-agent/api/document"
	"github.com/swiftbit/know-agent/api/knowledge"
	"github.com/swiftbit/know-agent/internal/svc"
	"github.com/swiftbit/know-agent/internal/trigger/consumer"
)

var ProviderSet = wire.NewSet(NewServer, NewHTTPServer)

type Server struct {
	HTTP               *rest.Server
	parseConsumer      *consumer.ParseDocumentConsumer
	buildIndexConsumer *consumer.BuildIndexConsumer
}

func NewServer(HTTP *rest.Server, parseConsumer *consumer.ParseDocumentConsumer, buildIndexConsumer *consumer.BuildIndexConsumer) *Server {
	return &Server{
		HTTP:               HTTP,
		parseConsumer:      parseConsumer,
		buildIndexConsumer: buildIndexConsumer,
	}
}

func (s *Server) Start() {
	s.parseConsumer.Start()
	s.buildIndexConsumer.Start()
	s.HTTP.Start()
}

func (s *Server) Stop() {
	s.HTTP.Stop()
}

func NewHTTPServer(svcCtx *svc.ServiceContext, docSrv document.HTTPServer, chatSrv chat.HTTPServer, knowledgeSrv knowledge.HTTPServer) *rest.Server {
	opts := []rest.RunOption{
		rest.WithCorsHeaders("*"),
		rest.WithCustomCors(func(header http.Header) {
			header.Add("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token,Authorization,Token,X-Token,X-User-Id,OS,Platform, Version")
			header.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS,PATCH")
			header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type")
		}, nil, "*"),
	}
	server := rest.MustNewServer(svcCtx.Config.Http, opts...)
	document.RegisterHandlers(server, svcCtx, docSrv)
	chat.RegisterHandlers(server, svcCtx, chatSrv)
	knowledge.RegisterHandlers(server, svcCtx, knowledgeSrv)
	return server
}
