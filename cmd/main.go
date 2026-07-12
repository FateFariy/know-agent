package main

import (
	"flag"
	"fmt"

	"github.com/zeromicro/go-zero/core/conf"

	"github.com/swiftbit/know-agent/internal/config"
)

var configFile = flag.String("f", "etc/config-dev.yaml", "the config file")

func main() {
	flag.Parse()

	var c *config.Config
	conf.MustLoad(*configFile, c)

	server := WireApp(c)
	defer server.HTTP.Stop()

	fmt.Printf("Starting HTTP server at %s:%d...\n", c.Http.Host, c.Http.Port)
	server.HTTP.Start()
}
