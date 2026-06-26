package parse

import (
	"context"
)

type Parser interface {
	Name() string
	Parse(ctx context.Context, bytesData []byte) (string, error)
}

type Registry struct {
	parsers        map[string]Parser
	fallbackParser Parser
}

func NewRegistry(fallbackParser Parser, parsers ...Parser) *Registry {
	registry := &Registry{
		parsers:        make(map[string]Parser),
		fallbackParser: fallbackParser,
	}
	for _, parser := range parsers {
		registry.Register(parser)
	}
	return registry
}

func (r *Registry) Register(parser Parser) {
	r.parsers[parser.Name()] = parser
}

func (r *Registry) Get(fileType string) (Parser, bool) {
	parser, ok := r.parsers[fileType]
	return parser, ok
}
