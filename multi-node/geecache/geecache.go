package geecache

import (
	"fmt"
	"log"
	"sync"
)

// Group 表示一个缓存命名空间，并包含了相关的数据加载和分发
type Group struct {
	name      string     // 缓存组的名称
	getter    Getter     // 数据加载器，用于从外部源获取数据
	mainCache cache      // 主缓存，使用 LRU 缓存策略
	peers     PeerPicker // 远程节点选择器，用于从远程节点获取缓存
}

// Getter 是一个接口，用于从外部获取缓存数据
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 是一个实现 Getter 接口的函数类型
type GetterFunc func(key string) ([]byte, error)

// Get 实现 Getter 接口方法
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex              // 用于保护并发访问缓存组的同步锁
	groups = make(map[string]*Group) // 缓存组的映射，按名称存储
)

// NewGroup 创建一个新的缓存组实例
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter") // 如果 Getter 为 nil，则抛出错误
	}
	mu.Lock()         // 加锁，保证线程安全
	defer mu.Unlock() // 解锁
	// 创建一个新的 Group 实例，并将其添加到缓存组映射中
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes}, // 创建主缓存
	}
	groups[name] = g
	return g
}

// GetGroup 返回通过 NewGroup 创建的缓存组，如果没有找到，则返回 nil
func GetGroup(name string) *Group {
	mu.RLock() // 加锁，允许多个读操作并发执行
	g := groups[name]
	mu.RUnlock() // 解锁
	return g
}

// Get 从缓存中获取指定键的值
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required") // 如果键为空，返回错误
	}

	// 尝试从主缓存中获取值
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit") // 如果缓存命中，打印日志
		return v, nil
	}

	// 如果缓存没有命中，从其他地方加载数据
	return g.load(key)
}

// RegisterPeers 注册一个 PeerPicker，用于选择远程节点
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once") // 防止多次注册 PeerPicker
	}
	g.peers = peers
}

// load 方法尝试从远程节点获取数据，若失败则从本地加载
func (g *Group) load(key string) (value ByteView, err error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok { // 从远程节点选择器中选择节点
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil // 如果从远程节点成功获取数据，返回
			}
			log.Println("[GeeCache] Failed to get from peer", err)
		}
	}

	// 如果远程节点获取失败，从本地加载数据
	return g.getLocally(key)
}

// populateCache 将数据添加到缓存中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// getLocally 从本地获取数据
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key) // 使用 Getter 从外部加载数据
	if err != nil {
		return ByteView{}, err // 如果加载数据失败，返回错误
	}
	value := ByteView{b: cloneBytes(bytes)} // 将数据包装成 ByteView 类型
	g.populateCache(key, value)             // 将数据存入缓存
	return value, nil
}

// getFromPeer 从远程节点获取数据
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key) // 从远程节点获取数据
	if err != nil {
		return ByteView{}, err // 如果获取失败，返回错误
	}
	return ByteView{b: bytes}, nil // 返回从远程节点获取的数据
}
