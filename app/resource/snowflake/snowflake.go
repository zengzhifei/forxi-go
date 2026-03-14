package snowflake

import (
	snowflakelib "github.com/bwmarrin/snowflake"
)

type Snowflake struct {
	node *snowflakelib.Node
}

// InitSnowflake 初始化雪花算法
// workerID: 机器ID (0-1023)
// datacenterID: 数据中心ID（保留参数，bwmarrin/snowflake只使用单一节点ID）
func InitSnowflake(workerID, datacenterID int64) (*Snowflake, error) {
	// bwmarrin/snowflake 使用单一节点ID (0-1023)
	// 我们将workerID和datacenterID组合成一个节点ID
	nodeID := (datacenterID << 5) | workerID
	if _node, err := snowflakelib.NewNode(nodeID); err != nil {
		return nil, err
	} else {
		return &Snowflake{node: _node}, nil
	}
}

// GenerateSnowflakeID 生成雪花ID
func (s *Snowflake) GenerateSnowflakeID() int64 {
	return s.node.Generate().Int64()
}
