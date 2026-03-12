package util

import (
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	node *snowflake.Node
	once sync.Once
)

// InitSnowflake 初始化雪花算法
// workerID: 机器ID (0-1023)
// datacenterID: 数据中心ID（保留参数，bwmarrin/snowflake只使用单一节点ID）
func InitSnowflake(workerID, datacenterID int64) error {
	var err error
	once.Do(func() {
		// bwmarrin/snowflake 使用单一节点ID (0-1023)
		// 我们将workerID和datacenterID组合成一个节点ID
		nodeID := (datacenterID << 5) | workerID
		node, err = snowflake.NewNode(nodeID)
	})
	return err
}

// GenerateSnowflakeID 生成雪花ID
func GenerateSnowflakeID() int64 {
	if node == nil {
		// 如果未初始化，使用默认节点1
		_ = InitSnowflake(1, 0)
	}
	return node.Generate().Int64()
}