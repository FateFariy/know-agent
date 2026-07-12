package vo

import (
	"github.com/duke-git/lancet/v2/strutil"
)

type KnowledgeScope = string

const (
	ScopeOperationRule   KnowledgeScope = "operation_rule"
	ScopeRobotStrategy   KnowledgeScope = "robot_strategy"
	ScopeDeployment      KnowledgeScope = "deployment"
	ScopeTroubleshooting KnowledgeScope = "troubleshooting"
	ScopeProduct         KnowledgeScope = "product"
	ScopeGeneral         KnowledgeScope = "general_document"
)

// KnowledgeScopeName 知识范围名称
func KnowledgeScopeName(scope KnowledgeScope) string {
	switch scope {
	case ScopeOperationRule:
		return "运营规则"
	case ScopeRobotStrategy:
		return "机器人策略"
	case ScopeDeployment:
		return "安装部署"
	case ScopeTroubleshooting:
		return "故障排查"
	case ScopeProduct:
		return "产品资料"
	default:
		return "通用文档"
	}
}

// KnowledgeScopeCode 知识范围代码
func KnowledgeScopeCode(text string) KnowledgeScope {
	switch {
	case strutil.ContainsAny(text, []string{"上线观察", "值班规则", "观察时长", "运营"}):
		return ScopeOperationRule
	case strutil.ContainsAny(text, []string{"机器人", "知识召回", "意图识别", "策略设计"}):
		return ScopeRobotStrategy
	case strutil.ContainsAny(text, []string{"安装", "部署", "默认密码", "访问地址"}):
		return ScopeDeployment
	case strutil.ContainsAny(text, []string{"故障", "排查", "异常", "检查顺序"}):
		return ScopeTroubleshooting
	case strutil.ContainsAny(text, []string{"产品简介", "核心特性", "技术规格", "产品概述"}):
		return ScopeProduct
	default:
		return ScopeGeneral
	}
}
