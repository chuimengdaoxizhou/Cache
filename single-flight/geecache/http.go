package geecache

import (
	"Cache/single-flight/consistenthash"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/" // 默认基础路径
	defaultReplicas = 50            // 默认副本数
)

// HTTPPool 实现了 PeerPicker 接口，用于管理 HTTP 对等节点池。
type HTTPPool struct {
	self        string                 // 当前节点的基础 URL，例如 "https://example.net:8000"
	basePath    string                 // 基础路径
	mu          sync.Mutex             // 保护 peers 和 httpGetters 的并发访问
	peers       *consistenthash.Map    // 使用一致性哈希算法管理的对等节点
	httpGetters map[string]*httpGetter // 存储每个节点的 HTTP 获取器，键是节点的 URL
}

// NewHTTPPool 初始化一个 HTTP 对等节点池。
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 记录日志，显示服务器名称
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 处理所有 HTTP 请求。
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 判断请求路径是否符合预期
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	// 解析请求的路径，应该为 /<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// 获取对应的缓存组
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 获取缓存值
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回缓存值
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// Set 更新节点池的对等节点列表。
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...) // 添加对等节点
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	// 为每个对等节点创建 HTTP 获取器
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer 根据 key 选择一个对等节点。
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 使用一致性哈希算法选择对等节点
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil) // 确保 HTTPPool 实现了 PeerPicker 接口

// httpGetter 用于从远程节点获取数据。
type httpGetter struct {
	baseURL string // 远程节点的基础 URL
}

// Get 从远程节点获取数据。
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v", // 构造请求 URL
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u) // 发起 HTTP GET 请求
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// 检查响应状态
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	// 读取响应体内容
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil // 返回数据
}

var _ PeerGetter = (*httpGetter)(nil) // 确保 httpGetter 实现了 PeerGetter 接口
