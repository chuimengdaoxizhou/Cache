package lru

import "container/list"

// Cache 是一个 LRU（最近最少使用）缓存。
// 该缓存不支持并发访问。
type Cache struct {
	maxBytes int64                    // 最大缓存字节数
	nbytes   int64                    // 当前缓存占用的字节数
	ll       *list.List               // 双向链表，用于实现 LRU 策略
	cache    map[string]*list.Element // 存储缓存的键值对映射
	// 可选的，当某个条目被移除时执行的回调函数
	OnEvicted func(key string, value Value)
}

// entry 表示缓存中的一项条目
type entry struct {
	key   string // 键
	value Value  // 值
}

// Value 是缓存值的接口，要求实现 Len 方法来返回值所占的字节数
type Value interface {
	Len() int // 返回值的字节长度
}

// New 是 Cache 的构造函数，创建一个新的缓存实例
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,                       // 设置最大字节数
		ll:        list.New(),                     // 创建双向链表
		cache:     make(map[string]*list.Element), // 创建缓存映射
		OnEvicted: onEvicted,                      // 设置回调函数
	}
}

// Add 向缓存中添加一个值
func (c *Cache) Add(key string, value Value) {
	// 如果缓存中已经存在该键，则更新其值
	if ele, ok := c.cache[key]; ok {
		// 将该元素移到链表的前面（表示最近访问）
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)                               // 获取该元素的值
		c.nbytes += int64(value.Len()) - int64(kv.value.Len()) // 更新字节数
		kv.value = value                                       // 更新值
	} else {
		// 否则，新加入一个元素
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len()) // 更新字节数
	}

	// 如果当前缓存的字节数超过了最大限制，则移除最旧的条目
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Get 查找缓存中某个键的值
func (c *Cache) Get(key string) (value Value, ok bool) {
	// 如果缓存中存在该键，则返回对应的值
	if ele, ok := c.cache[key]; ok {
		// 将该元素移到链表的前面
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry) // 获取该元素的值
		return kv.value, true    // 返回值
	}
	return // 如果没有找到，返回零值和 false
}

// RemoveOldest 移除链表中最旧的元素
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // 获取链表尾部的元素
	if ele != nil {
		c.ll.Remove(ele)                                       // 从链表中删除该元素
		kv := ele.Value.(*entry)                               // 获取元素的值
		delete(c.cache, kv.key)                                // 从缓存中删除该键
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 更新字节数
		// 如果设置了回调函数，则调用回调函数
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Len 返回缓存中条目的数量
func (c *Cache) Len() int {
	return c.ll.Len() // 返回链表中元素的个数
}
