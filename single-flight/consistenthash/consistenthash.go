package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 定义一个函数类型，将字节数组映射为 uint32 类型的哈希值
type Hash func(data []byte) uint32

// Map 结构体包含一致性哈希环的相关信息
type Map struct {
	hash     Hash           // 哈希函数
	replicas int            // 每个节点的虚拟节点数
	keys     []int          // 存储哈希值的切片，已排序
	hashMap  map[int]string // 存储哈希值与节点的映射关系
}

// New 创建并返回一个新的 Map 实例
func New(replicas int, fn Hash) *Map {
	// 创建 Map 实例并初始化相关字段
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	// 如果没有提供哈希函数，则使用 crc32.ChecksumIEEE 作为默认哈希函数
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 将一些节点添加到一致性哈希环中
func (m *Map) Add(keys ...string) {
	// 遍历每个节点，生成对应的虚拟节点
	for _, key := range keys {
		// 每个节点生成多个虚拟节点
		for i := 0; i < m.replicas; i++ {
			// 计算虚拟节点的哈希值
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 将虚拟节点的哈希值添加到 keys 切片中
			m.keys = append(m.keys, hash)
			// 将虚拟节点的哈希值与节点映射到 hashMap 中
			m.hashMap[hash] = key
		}
	}
	// 对哈希值进行排序，以保证一致性哈希环的顺序
	sort.Ints(m.keys)
}

// Get 根据传入的 key 获取哈希环中最接近的节点
func (m *Map) Get(key string) string {
	// 如果哈希环中没有节点，直接返回空字符串
	if len(m.keys) == 0 {
		return ""
	}

	// 计算 key 的哈希值
	hash := int(m.hash([]byte(key)))
	// 使用二分查找法查找第一个大于等于该哈希值的位置
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// 返回找到的哈希值对应的节点
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
