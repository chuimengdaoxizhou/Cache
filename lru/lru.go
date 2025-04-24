package lru

import (
	"container/list"
)

// Cache 是 LRU 缓存的核心结构体
type Cache struct {
	maxBytes  int64                         // 允许使用的最大内存（字节）
	nbytes    int64                         // 当前已使用的内存
	ll        *list.List                    // 双向链表，保存缓存的顺序（前：最近使用，后：最久未使用）
	cache     map[string]*list.Element      // 字典，键是字符串，值是双向链表中的节点指针
	OnEvicted func(key string, value Value) // 当某条记录被移除时的回调函数（可选）
}

// entry 是链表节点中存储的内容
type entry struct {
	key   string // 缓存键
	value Value  // 缓存值
}

// Value 是缓存中 value 的接口
// 只需要实现 Len() int 方法，用于计算占用内存大小
type Value interface {
	Len() int
}

// New 创建一个新的 Cache 实例
// maxBytes：最大允许使用内存；onEvicted：元素被移除时的回调
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),                     // 初始化链表
		cache:     make(map[string]*list.Element), // 初始化哈希表
		OnEvicted: onEvicted,
	}
}

// Get 查找缓存
// 步骤：1. 在 map 中查找；2. 找到后将节点移动到队尾（表示最近使用）
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele) // 移动到队尾
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest 移除最近最少使用的缓存项
// 即：删除链表队头节点，并更新 map 和已用内存
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // 获取队头元素
	if ele != nil {
		c.ll.Remove(ele) // 从链表中移除
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)                                // 从 map 中删除
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 更新当前内存使用量
		if c.OnEvicted != nil {                                // 如果设置了删除回调，调用
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add 向缓存中添加键值对
// 如果键存在，则更新值并将节点移到队尾
// 如果键不存在，则新建节点并插入队尾，同时更新 map 和已用内存
// 如果超过最大内存限制，则循环淘汰旧节点
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele) // 已存在则移动到队头
		kv := ele.Value.(*entry)
		// 更新内存大小差值
		c.nbytes += int64(len(key)) + int64(value.Len()) - int64(len(kv.key)) - int64(kv.value.Len())
		kv.key = key
		kv.value = value
	} else {
		// 不存在则新建 entry 节点
		ele := &entry{key, value}
		c.cache[key] = c.ll.PushFront(ele)               // 插入链表队头
		c.nbytes += int64(len(key)) + int64(value.Len()) // 增加内存使用
	}

	// 如果超出内存限制，循环移除最近最少使用节点
	for c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.RemoveOldest()
	}
}

// Len 返回当前缓存中条目数（不是字节数）
func (c *Cache) Len() int {
	return c.ll.Len()
}
