package entity

// Pipeline 管道
type Pipeline struct {
	ID          string       // 唯一标识符
	Name        string       // 名称
	Description string       // 描述信息
	Nodes       []NodeConfig // 节点配置列表，按执行顺序排列的节点配置
}
