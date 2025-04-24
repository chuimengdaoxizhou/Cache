package geecache

import (
	"fmt"
	"log"
	"sync"
)

// Group 代表一个缓存命名空间，每个 Group 拥有一个唯一的名字（name），
// 一个获取源数据的回调函数（getter），以及一个本地缓存（mainCache）。
type Group struct {
	name      string
	getter    Getter // 缓存未命中时获取源数据的回调接口
	mainCache cache  // 本地缓存
}

// Getter 接口，用于定义从数据源获取数据的方法
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 是一个实现了 Getter 接口的函数类型
type GetterFunc func(key string) ([]byte, error)

// Get 实现 Getter 接口的方法，使函数类型 GetterFunc 具备接口能力
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex              // 读写锁，保护 groups 的并发读写
	groups = make(map[string]*Group) // 全局 Group 注册表（map: groupName -> Group）
)

// NewGroup 创建一个新的缓存 Group 实例，并注册到全局 groups 中
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter") // 如果没有提供 Getter，直接 panic
	}
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes}, // 初始化本地缓存
	}
	groups[name] = g // 注册到全局 map 中
	return g
}

// GetGroup 用于根据名称获取已存在的 Group 实例
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 根据 key 获取缓存值
// 先从本地缓存中查找，未命中则调用 load 加载数据
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required") // key 不能为空
	}

	// 先从本地缓存中查找
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit") // 命中日志
		return v, nil
	}

	// 未命中，则调用 load 加载数据
	return g.load(key)
}

// load 是获取数据的入口（本地或远程），目前只实现了本地加载
func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

// getLocally 表示通过回调函数从数据源获取数据
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key) // 调用用户提供的 getter 获取源数据
	if err != nil {
		return ByteView{}, err
	}

	// 将源数据包装成只读的 ByteView 并添加到缓存中
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

// populateCache 将从源头获取的值添加到缓存中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
