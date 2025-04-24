package consistenthash

import (
	"hash/crc32" // 引入 crc32 哈希算法
	"sort"       // 引入排序包，用于排序哈希值
	"strconv"    // 引入 strconv 包，便于将整数转为字符串
)

// Hash 类型定义了哈希函数的接口，输入字节数组，输出 uint32 类型的哈希值
type Hash func(data []byte) uint32

// Map 结构体表示一致性哈希映射
// - hash：哈希函数，用于计算哈希值
// - replicas：每个真实节点的虚拟节点数量
// - keys：存储所有的虚拟节点哈希值，按顺序排列
// - hashMap：存储哈希值与真实节点的映射关系
type Map struct {
	hash     Hash
	replicas int
	keys     []int          // 排序后的哈希值列表
	hashMap  map[int]string // 哈希值到真实节点的映射
}

// New 函数创建并返回一个新的一致性哈希映射实例
// - replicas：每个真实节点的虚拟节点数量
// - fn：自定义的哈希函数，如果为 nil，则使用默认的 crc32.ChecksumIEEE 哈希函数
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	// 如果没有传入哈希函数，则使用默认的 crc32 哈希
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 方法将一个或多个节点添加到一致性哈希环中
// - 对于每个传入的节点，会创建 m.replicas 个虚拟节点，并计算虚拟节点的哈希值
// - 每个虚拟节点会被映射到实际节点，并存储在 hashMap 中
// - 所有虚拟节点的哈希值会按升序排序，确保节点位置的确定性
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			// 计算虚拟节点的哈希值，使用虚拟节点名称（虚拟节点名称由 i 和 key 组成）
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 将哈希值添加到 keys 列表中
			m.keys = append(m.keys, hash)
			// 在 hashMap 中保存虚拟节点哈希值到实际节点的映射
			m.hashMap[hash] = key
		}
	}
	// 对所有虚拟节点的哈希值进行排序
	sort.Ints(m.keys)
}

// Get 方法根据给定的 key 获取离该 key 最近的节点
// - 首先计算 key 的哈希值
// - 使用二分查找找到哈希值大于等于 key 哈希值的最小虚拟节点的位置
// - 返回该虚拟节点对应的真实节点
func (m *Map) Get(key string) string {
	// 如果哈希环为空，返回空字符串
	if len(m.keys) == 0 {
		return ""
	}

	// 计算 key 的哈希值
	hash := int(m.hash([]byte(key)))
	// 使用二分查找在 m.keys 中找到第一个大于等于 key 哈希值的位置
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// 返回与计算出的虚拟节点对应的真实节点
	// 通过取余运算，确保索引在 keys 数组长度范围内
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
