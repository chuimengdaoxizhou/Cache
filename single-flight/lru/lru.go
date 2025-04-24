package lru

import "container/list"

// Cache 是一个 LRU 缓存实现，不支持并发访问。
type Cache struct {
	maxBytes  int64                         // 缓存的最大字节数
	nbytes    int64                         // 当前缓存占用的字节数
	ll        *list.List                    // 双向链表，用于存储缓存条目
	cache     map[string]*list.Element      // 用于存储缓存条目的哈希表，key 是缓存的键，值是链表中的元素
	OnEvicted func(key string, value Value) // 可选，当条目被清除时执行的回调函数
}

// entry 表示缓存中的一条记录
type entry struct {
	key   string // 键
	value Value  // 值
}

// Value 是缓存条目的值，必须实现 Len 方法，用于计算占用的字节数
type Value interface {
	Len() int
}

// New 是 Cache 的构造函数，返回一个新的 Cache 实例
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),                     // 初始化双向链表
		cache:     make(map[string]*list.Element), // 初始化哈希表
		OnEvicted: onEvicted,                      // 设置当条目被清除时执行的回调函数
	}
}

// Add 向缓存中添加一个值
func (c *Cache) Add(key string, value Value) {
	// 如果缓存中已经有该键
	if ele, ok := c.cache[key]; ok {
		// 将该条目移动到链表的前端
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		// 更新缓存的字节数
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		// 更新条目的值
		kv.value = value
	} else {
		// 如果缓存中没有该键，创建一个新的条目
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		// 更新缓存的字节数
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	// 如果缓存的字节数超过了最大字节数，删除最旧的条目
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Get 查找并返回缓存中键对应的值
func (c *Cache) Get(key string) (value Value, ok bool) {
	// 如果缓存中存在该键
	if ele, ok := c.cache[key]; ok {
		// 将该条目移动到链表的前端
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		// 返回条目的值
		return kv.value, true
	}
	return
}

// RemoveOldest 删除链表中最旧的条目（即尾部的条目）
func (c *Cache) RemoveOldest() {
	// 获取链表的最后一个元素（最旧的条目）
	ele := c.ll.Back()
	if ele != nil {
		// 从链表中移除该条目
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// 从缓存中删除该条目
		delete(c.cache, kv.key)
		// 更新缓存的字节数
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		// 如果定义了回调函数，执行回调
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Len 返回缓存中条目的数量
func (c *Cache) Len() int {
	return c.ll.Len()
}
