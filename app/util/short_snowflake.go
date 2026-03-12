package util

import (
	"errors"
	"sync"
	"time"
)

const (
	// 时间戳起始点 2023-01-01 00:00:00 UTC
	shortTwepoch int64 = 1672502400
	
	// 各部分位数
	shortWorkerIDBits      uint8 = 2  // 机器ID：2位，最多4台
	shortDatacenterIDBits  uint8 = 2  // 数据中心ID：2位，最多4个机房
	shortSequenceBits      uint8 = 4  // 序列号：4位，每秒最多16个ID
	
	// 各部分最大值
	maxShortWorkerID      int64 = -1 ^ (-1 << shortWorkerIDBits)      // 3
	maxShortDatacenterID  int64 = -1 ^ (-1 << shortDatacenterIDBits)  // 3
	maxShortSequence      int64 = -1 ^ (-1 << shortSequenceBits)      // 15
	
	// 各部分位移
	shortWorkerIDShift      uint8 = shortSequenceBits                                          // 4
	shortDatacenterIDShift  uint8 = shortSequenceBits + shortWorkerIDBits                      // 6
	shortTimestampShift     uint8 = shortSequenceBits + shortWorkerIDBits + shortDatacenterIDBits  // 8
	
	// 默认时钟回拨容忍度：2秒
	defaultTimeOffset int64 = 2
)

// ShortSnowflake 短雪花算法ID生成器
// 适合最多4个机房，单机房4台实例，生成效率要求不高的场景
// 生成的ID约11-13位数字
type ShortSnowflake struct {
	mu            sync.Mutex
	lastTimestamp int64
	workerID      int64
	datacenterID  int64
	sequence      int64
	timeOffset    int64  // 允许的时钟回拨秒数
}

var (
	shortSnowflake *ShortSnowflake
	shortOnce      sync.Once
)

// InitShortSnowflake 初始化短雪花算法
// workerID: 机器ID (0-3)
// datacenterID: 数据中心ID (0-3)
func InitShortSnowflake(workerID, datacenterID int64) error {
	if workerID > maxShortWorkerID || workerID < 0 {
		return errors.New("worker ID must be between 0 and 3")
	}
	if datacenterID > maxShortDatacenterID || datacenterID < 0 {
		return errors.New("datacenter ID must be between 0 and 3")
	}
	
	shortOnce.Do(func() {
		shortSnowflake = &ShortSnowflake{
			workerID:     workerID,
			datacenterID: datacenterID,
			timeOffset:   defaultTimeOffset,
		}
	})
	return nil
}

// GenerateShortSnowflakeID 生成短雪花ID
func GenerateShortSnowflakeID() int64 {
	if shortSnowflake == nil {
		// 如果未初始化，使用默认值
		_ = InitShortSnowflake(1, 0)
	}
	return shortSnowflake.NextID()
}

// NextID 获取下一个ID
func (s *ShortSnowflake) NextID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 获取当前秒级时间戳
	timestamp := time.Now().Unix()
	
	// 时钟回拨处理
	if timestamp < s.lastTimestamp {
		offset := s.lastTimestamp - timestamp
		if offset <= s.timeOffset {
			// 容忍小范围回拨（如NTP校时）
			timestamp = s.lastTimestamp
		} else {
			// 超出容忍范围，报错
			panic(errors.New("clock moved backwards, refusing to generate id"))
		}
	}
	
	// 同一秒内，序列号递增
	if timestamp == s.lastTimestamp {
		s.sequence = (s.sequence + 1) & maxShortSequence
		if s.sequence == 0 {
			// 序列号用完，等待下一秒
			timestamp = s.waitNextSecond(timestamp)
		}
	} else {
		// 新的一秒，序列号重置为0
		s.sequence = 0
	}
	
	s.lastTimestamp = timestamp
	
	// 组装ID：时间戳(秒) + 数据中心ID + 机器ID + 序列号
	id := ((timestamp - shortTwepoch) << shortTimestampShift) |
		(s.datacenterID << shortDatacenterIDShift) |
		(s.workerID << shortWorkerIDShift) |
		s.sequence
	
	return id
}

// waitNextSecond 等待下一秒
func (s *ShortSnowflake) waitNextSecond(lastTimestamp int64) int64 {
	timestamp := time.Now().Unix()
	for timestamp <= lastTimestamp {
		time.Sleep(time.Millisecond * 100) // 短暂休眠
		timestamp = time.Now().Unix()
	}
	
	// 再次检查时钟回拨
	if timestamp < lastTimestamp {
		panic(errors.New("clock moved backwards after waiting"))
	}
	
	return timestamp
}

// ParseShortSnowflakeID 解析短雪花ID
func ParseShortSnowflakeID(id int64) (timestamp, datacenterID, workerID, sequence int64) {
	timestamp = (id >> shortTimestampShift) + shortTwepoch
	datacenterID = (id >> shortDatacenterIDShift) & maxShortDatacenterID
	workerID = (id >> shortWorkerIDShift) & maxShortWorkerID
	sequence = id & maxShortSequence
	return
}