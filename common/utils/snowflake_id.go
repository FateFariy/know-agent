package utils

import "github.com/bwmarrin/snowflake"

var Node, _ = snowflake.NewNode(1)

func GetSnowflakeNextID() int64 {
	return Node.Generate().Int64()
}
