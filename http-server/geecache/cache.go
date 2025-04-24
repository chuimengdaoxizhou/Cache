package geecache

import (
	"Cache/http-server/geecache/lru"
	"sync"
)

// cache 封装了 lru.Cache，并添加了互斥锁以保证并发安全
type cache struct {
	mu         sync.Mutex // 互斥锁，用于并发控制
	lru        *lru.Cache // 实际的 LRU 缓存结构
	cacheBytes int64      // 允许使用的最大内存（字节）
}

// add 向缓存中添加键值对
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()         // 加锁，确保并发安全
	defer c.mu.Unlock() // 函数退出时自动解锁

	// 延迟初始化 LRU 缓存实例
	if c.lru == nil {
		// 不设置淘汰回调（传 nil）
		c.lru = lru.New(c.cacheBytes, nil)
	}
	// 添加键值对
	c.lru.Add(key, value)
}

// get 从缓存中根据 key 获取对应的值
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()         // 加锁，确保并发安全
	defer c.mu.Unlock() // 函数退出时自动解锁

	// 如果缓存未初始化，直接返回空值
	if c.lru == nil {
		return
	}

	// 查询缓存
	if v, ok := c.lru.Get(key); ok {
		// 类型断言为 ByteView 并返回
		return v.(ByteView), ok
	}

	// 没找到，返回默认值
	return
}
