package geecache

import (
	"Cache/multi-node/lru" // 引入 lru 包，用于实现 LRU 缓存
	"sync"                 // 引入 sync 包，用于同步操作
)

// cache 结构体表示一个缓存，使用了 LRU 缓存策略。
// 它包含了一个同步锁、LRU 缓存实例以及缓存大小。
type cache struct {
	mu         sync.Mutex // 用于保证缓存的线程安全
	lru        *lru.Cache // LRU 缓存实例
	cacheBytes int64      // 缓存的最大字节数
}

// add 方法将一个键值对添加到缓存中。
// 它首先会检查是否初始化了 LRU 缓存，如果没有，则创建一个新的 LRU 缓存实例。
// 然后将指定的键值对添加到缓存中。
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()         // 锁定缓存，确保线程安全
	defer c.mu.Unlock() // 解锁缓存

	// 如果缓存为空，初始化一个新的 LRU 缓存
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}

	// 将键值对添加到 LRU 缓存中
	c.lru.Add(key, value)
}

// get 方法从缓存中获取指定键的值。
// 它会返回缓存中存储的值（如果存在），否则返回默认值。
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()         // 锁定缓存，确保线程安全
	defer c.mu.Unlock() // 解锁缓存

	// 如果缓存为空，直接返回
	if c.lru == nil {
		return
	}

	// 从 LRU 缓存中获取值
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok // 如果存在，返回值和存在标志
	}

	// 如果缓存中没有找到，返回默认值
	return
}
