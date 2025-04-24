package geecache

import (
	pb "Cache/proto-buf/geecache/geecachepb"
	"Cache/proto-buf/geecache/singleflight"
	"fmt"
	"log"
	"sync"
)

// Group 是一个缓存命名空间和相关数据的载体
type Group struct {
	name      string              // 组名
	getter    Getter              // 数据加载器
	mainCache cache               // 主缓存
	peers     PeerPicker          // 远程节点选择器
	loader    *singleflight.Group // 单次请求组，确保每个键值请求只会加载一次
}

// Getter 用于从外部源加载数据
type Getter interface {
	Get(key string) ([]byte, error) // 从外部源获取数据
}

// GetterFunc 实现了 Getter 接口的函数类型
type GetterFunc func(key string) ([]byte, error)

// Get 实现了 Getter 接口的 Get 方法
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	mu     sync.RWMutex              // 用于并发读写的锁
	groups = make(map[string]*Group) // 存储所有创建的 Group
)

// NewGroup 创建一个新的 Group 实例
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter") // 如果 Getter 为空，抛出错误
	}
	mu.Lock()
	defer mu.Unlock()

	// 创建一个新的 Group 并初始化
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes}, // 初始化缓存
		loader:    &singleflight.Group{},         // 使用 singleflight.Group 防止重复请求
	}

	// 将创建的 Group 注册到全局的 groups 中
	groups[name] = g
	return g
}

// GetGroup 返回之前创建的命名 Group，如果没有则返回 nil
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name] // 从 groups 中获取指定的 Group
	mu.RUnlock()
	return g
}

// Get 从缓存中获取指定键的值
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required") // 键不能为空
	}

	// 尝试从主缓存中获取数据
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit") // 如果命中缓存，打印日志
		return v, nil
	}

	// 如果没有命中缓存，从外部源加载数据
	return g.load(key)
}

// RegisterPeers 注册远程节点选择器
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once") // 确保 PeerPicker 只被注册一次
	}
	g.peers = peers
}

// load 加载数据，确保每个 key 只会请求一次
func (g *Group) load(key string) (value ByteView, err error) {
	// 使用 singleflight.Group 确保每个键只会请求一次
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// 如果有远程节点，尝试从远程节点获取数据
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				// 尝试从远程 peer 获取数据
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}

		// 如果远程获取失败，从本地加载数据
		return g.getLocally(key)
	})

	// 如果没有错误，返回获取的数据
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

// populateCache 将数据添加到缓存中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// getLocally 从本地加载数据
func (g *Group) getLocally(key string) (ByteView, error) {
	// 使用 getter 从外部源获取数据
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	// 将获取的数据封装成 ByteView 并缓存
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

// getFromPeer 从远程节点获取数据
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	// 通过 peer 调用远程接口获取数据
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
