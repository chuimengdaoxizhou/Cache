package geecache

import (
	"Cache/multi-node/consistenthash"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/" // 默认的请求路径前缀
	defaultReplicas = 50            // 一致性哈希的虚拟节点数
)

// HTTPPool 实现了 PeerPicker 接口，用于管理一个 HTTP 连接池。
type HTTPPool struct {
	self        string                 // 当前节点的地址，例如 "http://localhost:8000"
	basePath    string                 // 请求路径的基础部分，默认为 "/_geecache/"
	mu          sync.Mutex             // 保护 peers 和 httpGetters 的并发访问
	peers       *consistenthash.Map    // 一致性哈希环，映射每个 key 到一个 peer
	httpGetters map[string]*httpGetter // 存储所有 HTTP peer 的 getter，按 peer 地址进行键值映射
}

// NewHTTPPool 初始化一个 HTTP pool。
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,            // 当前节点的地址
		basePath: defaultBasePath, // 使用默认基路径
	}
}

// Log 打印服务器日志，格式为 "[Server <self>] <message>"。
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 处理所有 HTTP 请求。
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 判断请求路径是否以 basePath 开头
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	// 提取 groupName 和 key
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// 获取对应的 Group
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 从 Group 获取数据
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回缓存的 ByteView 数据
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// Set 更新当前池中的 peers 列表。
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()         // 加锁，保护共享数据
	defer p.mu.Unlock() // 解锁
	// 初始化一致性哈希环，并将所有 peers 添加进去
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	// 初始化 httpGetters 字典，存储每个 peer 的 httpGetter 实例
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer 根据 key 选择一个 peer。
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()         // 加锁，保护共享数据
	defer p.mu.Unlock() // 解锁
	// 使用一致性哈希来选择一个 peer
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// 确保 HTTPPool 实现了 PeerPicker 接口
var _ PeerPicker = (*HTTPPool)(nil)

// httpGetter 用于通过 HTTP 从远程 peer 获取数据。
type httpGetter struct {
	baseURL string // 远程 peer 的基本 URL
}

// Get 从远程 peer 获取数据。
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	// 构建请求 URL
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group), // URL 编码 group 和 key
		url.QueryEscape(key),
	)
	// 发送 HTTP GET 请求
	res, err := http.Get(u)
	if err != nil {
		return nil, err // 请求失败，返回错误
	}
	defer res.Body.Close()

	// 如果返回的状态码不是 200 OK，则返回错误
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	// 读取响应体
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

// 确保 httpGetter 实现了 PeerGetter 接口
var _ PeerGetter = (*httpGetter)(nil)
