package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 是一个将字节数组映射为 uint32 的函数类型
type Hash func(data []byte) uint32

// Map 结构体包含所有已哈希的键
type Map struct {
	hash     Hash           // 哈希函数
	replicas int            // 每个节点的虚拟节点数
	keys     []int          // 排序后的哈希值
	hashMap  map[int]string // 哈希值到节点的映射
}

// New 创建一个新的 Map 实例
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,             // 设置虚拟节点数
		hash:     fn,                   // 设置哈希函数
		hashMap:  make(map[int]string), // 初始化哈希映射
	}
	// 如果没有提供哈希函数，则使用 crc32 作为默认哈希函数
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 向哈希中添加一些节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 每个节点会添加多个虚拟节点
		for i := 0; i < m.replicas; i++ {
			// 生成虚拟节点的哈希值
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 将哈希值添加到 keys 切片
			m.keys = append(m.keys, hash)
			// 将哈希值映射到节点
			m.hashMap[hash] = key
		}
	}
	// 对所有的哈希值进行排序
	sort.Ints(m.keys)
}

// Get 获取与提供的键最接近的节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return "" // 如果没有节点，返回空字符串
	}

	// 计算提供的键的哈希值
	hash := int(m.hash([]byte(key)))
	// 使用二分查找找到第一个大于或等于该哈希值的位置
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// 返回对应的节点，如果索引越界则循环使用第一个节点
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
