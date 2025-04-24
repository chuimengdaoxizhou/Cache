package single_node

import (
	"fmt"
	"log"
	"sync"
)

// Group 是 GeeCache 最核心的数据结构，
// 一个 Group 可以看做是一个缓存空间的命名空间（namespace），每个 Group 拥有唯一名称 name
// 以及对应的 Getter（当缓存未命中时，获取数据的回调函数）和主缓存 mainCache。
type Group struct {
	name      string
	getter    Getter // 缓存未命中时获取源数据的回调函数
	mainCache cache  // 本地缓存结构
}

// Getter 接口定义了获取数据的方法
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 是一个函数类型，实现了 Getter 接口
type GetterFunc func(key string) ([]byte, error)

// Get 实现了 Getter 接口，直接调用函数
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex              // 读写锁，保护 groups 并发访问
	groups = make(map[string]*Group) // 全局注册的缓存组
)

// NewGroup 用于创建一个新的 Group 实例
// 参数：
// - name：Group 名称
// - cacheBytes：缓存可使用的最大内存
// - getter：回调函数，当缓存未命中时调用获取源数据
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter") // 不允许为空
	}
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// GetGroup 根据名称获取已存在的 Group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 根据 key 获取缓存值
// 首先尝试从缓存中查找，查找不到则调用 load 方法获取数据
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit") // 缓存命中日志
		return v, nil
	}

	// 未命中则加载数据
	return g.load(key)
}

// load 方法尝试从本地或远程加载数据（当前只实现本地加载）
func (g *Group) load(key string) (value ByteView, err error) {
	// 本地加载（也可以扩展远程加载）
	return g.getLocally(key)
}

// getLocally 调用用户提供的回调函数，从源头获取数据
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	// 将获取的数据封装为 ByteView 并加入缓存
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)

	return value, nil
}

// populateCache 将键值对添加到缓存中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
