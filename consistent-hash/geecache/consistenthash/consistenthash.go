package consistenthash

import (
	"hash/crc32"
	"sort"
)

// Hash 定义哈希函数的类型，输入是字节数组，输出是 uint32 的哈希值
type Hash func(data []byte) uint32

// Map 是一致性哈希的核心数据结构
type Map struct {
	hash     Hash           // 哈希函数
	replicas int            // 每个真实节点对应的虚拟节点数量
	keys     []int          // 哈希环上所有的虚拟节点（排序后的哈希值）
	hashMap  map[int]string // 哈希值与真实节点的映射表
}

// New 创建一个新的一致性哈希 Map
// 参数 replicas 表示每个真实节点有多少个虚拟节点
// 参数 fn 是用户自定义的哈希函数（可选）
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}

	// 如果用户没有提供哈希函数，使用 crc32.ChecksumIEEE 默认实现
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}

	return m
}

// Add 向哈希环中添加节点（支持添加一个或多个真实节点）
// 每个真实节点会创建 m.replicas 个虚拟节点，避免数据倾斜
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			// 创建虚拟节点名，例如：0nodeA、1nodeA...
			virtualNodeName := []byte(string(i) + key)
			hash := int(m.hash(virtualNodeName)) // 计算虚拟节点哈希值

			m.keys = append(m.keys, hash) // 放入哈希环
			m.hashMap[hash] = key         // 记录虚拟节点映射的真实节点
		}
	}
	sort.Ints(m.keys) // 对哈希值排序，形成环状结构（从小到大）
}
