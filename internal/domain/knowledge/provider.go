package knowledge

import (
	"github.com/google/wire"

	knowledgelogic "github.com/swiftbit/know-agent/internal/domain/knowledge/logic"
	"github.com/swiftbit/know-agent/internal/domain/knowledge/logic/route"
)

var ProviderSet = wire.NewSet(
	route.NewKnowledgeRouteImpl,
	wire.Bind(new(route.KnowledgeRouter), new(*route.KnowledgeRouteImpl)),
	knowledgelogic.NewKnowledgeLogicImpl,
	wire.Bind(new(knowledgelogic.KnowledgeLogic), new(*knowledgelogic.KnowledgeLogicImpl)),
	knowledgelogic.NewKnowledgeConfigLogicImpl,
	wire.Bind(new(knowledgelogic.KnowledgeConfigLogic), new(*knowledgelogic.KnowledgeConfigLogicImpl)),
	ProvideKnowledgeOptions,
)

// ProvideKnowledgeOptions 提供知识路由的可选项（目前为空），
// 供 NewKnowledgeRouteImpl 消费。
func ProvideKnowledgeOptions() []route.Option {
	return nil
}
