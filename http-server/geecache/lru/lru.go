package lru

import "container/list"

// Cache 是一个最近最少使用（LRU）缓存。它不是并发安全的。
type Cache struct {
	maxBytes  int64                         // 允许使用的最大内存（字节）
	nbytes    int64                         // 当前已使用的内存
	ll        *list.List                    // 双向链表，用于记录元素访问顺序
	cache     map[string]*list.Element      // 字典，键是字符串，值是链表节点指针
	OnEvicted func(key string, value Value) // 可选的回调函数，当某条记录被移除时执行
}

// entry 是双向链表中存储的数据类型
type entry struct {
	key   string // 键
	value Value  // 值，实现了 Value 接口
}

// Value 接口，值需要实现 Len 方法，用于计算占用的字节数
type Value interface {
	Len() int
}

// New 是 Cache 的构造函数
// maxBytes 指定缓存占用的最大字节数
// onEvicted 是一个回调函数，当某个键值对被淘汰时调用
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Add 向缓存中添加一个键值对
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		// 如果 key 已存在，则移动到链表头部（表示最近访问）
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		// 更新内存使用大小（新值的长度减去旧值的长度）
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		// 如果 key 不存在，创建新的 entry 并插入链表头
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		// 增加总内存大小（包括 key 和 value）
		c.nbytes += int64(len(key)) + int64(value.Len())
	}

	// 如果超出最大内存限制，则移除最旧的元素
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Get 查找缓存中的键 key 对应的值
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 如果找到，移动到链表头部，并返回值
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	// 否则返回 nil 和 false
	return
}

// RemoveOldest 移除最旧的元素（链表尾部的元素）
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		// 从链表中移除
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// 从字典中删除
		delete(c.cache, kv.key)
		// 更新内存使用
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		// 如果设置了回调函数，则调用
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Len 返回缓存中当前的键值对数量
func (c *Cache) Len() int {
	return c.ll.Len()
}
