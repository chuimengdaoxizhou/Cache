package geecache

import (
	"Cache/proto-buf/geecache/lru"
	"sync"
)

type cache struct {
	mu         sync.Mutex // 用于保护并发访问
	lru        *lru.Cache // LRU 缓存
	cacheBytes int64      // 缓存的最大字节数
}

// add 向缓存中添加一个键值对
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()         // 上锁，防止并发访问时出现数据竞争
	defer c.mu.Unlock() // 函数退出时解锁

	// 如果 LRU 缓存为空，则创建一个新的 LRU 缓存
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}

	// 将键值对添加到 LRU 缓存中
	c.lru.Add(key, value)
}

// get 从缓存中获取一个值
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()         // 上锁，防止并发访问时出现数据竞争
	defer c.mu.Unlock() // 函数退出时解锁

	// 如果 LRU 缓存为空，直接返回
	if c.lru == nil {
		return
	}

	// 从 LRU 缓存中获取键对应的值
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok // 将缓存中的值转换为 ByteView 并返回
	}

	return // 如果没有找到，返回默认值
}
