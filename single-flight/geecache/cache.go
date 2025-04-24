package geecache

import (
	"Cache/single-flight/lru"
	"sync"
)

// cache 是一个缓存结构，包含一个 LRU 缓存和缓存大小限制。
type cache struct {
	mu         sync.Mutex // 用于保护缓存操作的锁
	lru        *lru.Cache // LRU 缓存
	cacheBytes int64      // 缓存的最大字节数
}

// add 将 key 和 value 添加到缓存中
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()         // 获取锁
	defer c.mu.Unlock() // 解锁
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil) // 如果 LRU 缓存尚未创建，初始化它
	}
	c.lru.Add(key, value) // 向 LRU 缓存中添加 key 和 value
}

// get 根据 key 获取缓存中的值
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()         // 获取锁
	defer c.mu.Unlock() // 解锁
	if c.lru == nil {
		return
	}

	// 从 LRU 缓存中获取数据，如果存在，返回缓存中的值
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}

	return // 如果缓存中没有该 key，则返回零值
}
