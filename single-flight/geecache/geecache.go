package geecache

import (
	"Cache/single-flight/singleflight"
	"fmt"
	"log"
	"sync"
)

// Group 是一个缓存命名空间和关联数据的容器。
type Group struct {
	name      string              // 缓存组的名称
	getter    Getter              // 数据加载器接口
	mainCache cache               // 主缓存
	peers     PeerPicker          // 对等节点选择器
	loader    *singleflight.Group // 用于确保每个 key 只被加载一次
}

// Getter 用于加载数据的方法接口。
type Getter interface {
	Get(key string) ([]byte, error) // 根据 key 获取数据
}

// GetterFunc 是 Getter 接口的函数实现。
type GetterFunc func(key string) ([]byte, error)

// Get 实现 Getter 接口的 Get 方法。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key) // 调用传入的函数
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group) // 存储缓存组
)

// NewGroup 创建一个新的 Group 实例。
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter") // 如果传入的 getter 为 nil，抛出错误
	}
	mu.Lock()
	defer mu.Unlock()
	// 创建 Group 实例
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes}, // 初始化缓存
		loader:    &singleflight.Group{},         // 初始化 singleflight.Group
	}
	// 将新创建的 Group 注册到全局的 groups 中
	groups[name] = g
	return g
}

// GetGroup 根据名称返回之前创建的缓存组，如果没有找到返回 nil。
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 根据 key 从缓存中获取值。
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required") // 如果 key 为空，返回错误
	}

	// 尝试从主缓存中获取数据
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit") // 缓存命中
		return v, nil
	}

	// 如果缓存未命中，则加载数据
	return g.load(key)
}

// RegisterPeers 注册 PeerPicker 用于选择远程对等节点。
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once") // 防止多次注册
	}
	g.peers = peers // 设置对等节点选择器
}

// load 从远程或本地加载数据
func (g *Group) load(key string) (value ByteView, err error) {
	// 使用 singleflight.Group 来确保每个 key 只会被加载一次，不管有多少并发请求。
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// 如果有远程对等节点，尝试从远程节点获取数据
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil // 如果从远程节点获取成功，返回结果
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}

		// 如果远程获取失败，尝试本地加载
		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil // 成功返回加载的数据
	}
	return // 返回错误
}

// populateCache 将数据放入缓存
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value) // 向主缓存中添加数据
}

// getLocally 从本地获取数据
func (g *Group) getLocally(key string) (ByteView, error) {
	// 调用 Getter 获取数据
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err // 如果获取失败，返回错误
	}
	value := ByteView{b: cloneBytes(bytes)} // 封装成 ByteView
	g.populateCache(key, value)             // 将数据缓存到主缓存
	return value, nil                       // 返回获取的数据
}

// getFromPeer 从远程对等节点获取数据
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key) // 从远程节点请求数据
	if err != nil {
		return ByteView{}, err // 如果失败，返回错误
	}
	return ByteView{b: bytes}, nil // 返回数据
}
