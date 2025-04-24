package lru

import "container/list"

// Cache 是一个实现 LRU（最近最少使用）缓存的结构体。
// 注意：此缓存不是并发安全的。
type Cache struct {
	maxBytes int64                    // 缓存最大字节数
	nbytes   int64                    // 当前缓存已使用的字节数
	ll       *list.List               // 双向链表，用于记录元素的顺序
	cache    map[string]*list.Element // 存储缓存数据的映射，key 对应链表中的元素
	// 当缓存条目被移除时，执行的回调函数（可选）
	OnEvicted func(key string, value Value)
}

// entry 表示缓存中的一个条目，包含 key 和 value
type entry struct {
	key   string // 键
	value Value  // 值
}

// Value 接口用于获取值的字节大小
// 通过 Len() 方法来计算值所占的字节数
type Value interface {
	Len() int
}

// New 是 Cache 的构造函数，用于创建一个新的 LRU 缓存实例
// - maxBytes：缓存的最大字节数
// - onEvicted：当缓存条目被移除时，执行的回调函数
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),                     // 创建一个新的双向链表
		cache:     make(map[string]*list.Element), // 创建一个映射，用于存储缓存
		OnEvicted: onEvicted,                      // 设置回调函数
	}
}

// Add 方法用于将一个值添加到缓存中
// - 如果缓存中已存在该键，则更新该值并将其移到链表的前面
// - 如果缓存中不存在该键，则将其插入链表的前面，并更新缓存的字节数
func (c *Cache) Add(key string, value Value) {
	// 如果缓存中已有该键
	if ele, ok := c.cache[key]; ok {
		// 移动该元素到链表的前面
		c.ll.MoveToFront(ele)
		// 获取元素中的 entry
		kv := ele.Value.(*entry)
		// 更新缓存的字节数，替换旧值的字节数
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		// 更新值
		kv.value = value
	} else {
		// 如果缓存中没有该键，创建新的缓存条目并插入到链表的前面
		ele := c.ll.PushFront(&entry{key, value})
		// 将新条目添加到 cache 映射中
		c.cache[key] = ele
		// 更新缓存的字节数
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	// 如果缓存的字节数超过了最大字节数，则删除最旧的条目
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Get 方法用于查找并返回缓存中的值
// - 如果该键存在，返回对应的值并将该键的条目移到链表前面
// - 如果该键不存在，返回 false
func (c *Cache) Get(key string) (value Value, ok bool) {
	// 如果缓存中存在该键
	if ele, ok := c.cache[key]; ok {
		// 将该元素移到链表的前面
		c.ll.MoveToFront(ele)
		// 获取元素中的 entry
		kv := ele.Value.(*entry)
		// 返回值
		return kv.value, true
	}
	// 如果缓存中没有该键，返回空值和 false
	return
}

// RemoveOldest 方法用于移除缓存中最旧的条目
// 通常用来删除超过最大字节数限制的条目
func (c *Cache) RemoveOldest() {
	// 获取链表中的最后一个元素（最旧的元素）
	ele := c.ll.Back()
	if ele != nil {
		// 从链表中删除该元素
		c.ll.Remove(ele)
		// 获取元素中的 entry
		kv := ele.Value.(*entry)
		// 从 cache 中删除该条目
		delete(c.cache, kv.key)
		// 更新缓存的字节数
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		// 如果设置了回调函数，执行它
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Len 方法返回当前缓存中条目的数量
func (c *Cache) Len() int {
	return c.ll.Len() // 返回链表的长度，表示缓存中有多少条数据
}
