package geecache

import (
	"Cache/proto-buf/geecache/consistenthash"
	pb "Cache/proto-buf/geecache/geecachepb"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
)

const (
	defaultBasePath = "/_geecache/" // 默认的基础路径
	defaultReplicas = 50            // 默认的副本数
)

// HTTPPool 实现了 PeerPicker 接口，用于处理 HTTP 请求的节点池。
type HTTPPool struct {
	self        string                 // 当前节点的 URL，例如 "https://example.net:8000"
	basePath    string                 // 基础路径，例如 "/_geecache/"
	mu          sync.Mutex             // 用于保护 peers 和 httpGetters 的锁
	peers       *consistenthash.Map    // 哈希环，用于根据 key 选择节点
	httpGetters map[string]*httpGetter // 存储节点的 httpGetter，按节点 URL 索引
}

// NewHTTPPool 初始化一个 HTTP 节点池
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 用于打印带有服务器名称的日志信息
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 处理所有的 HTTP 请求
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 检查请求路径是否以 basePath 开头
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	// 从路径中提取 groupName 和 key
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

	// 从缓存中获取数据
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 将值写入响应体，并以 proto 消息格式返回
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

// Set 更新节点池中的节点列表
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 使用一致性哈希来管理节点
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	// 为每个节点创建一个 httpGetter
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer 根据 key 选择一个远程节点
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// 根据一致性哈希算法选择一个节点
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// httpGetter 实现了 PeerGetter 接口，用于从远程节点获取数据
type httpGetter struct {
	baseURL string // 远程节点的基本 URL
}

// Get 从远程节点获取数据
func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	// 构建请求的 URL
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)

	// 发送 HTTP GET 请求
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// 如果响应状态码不是 200 OK，则返回错误
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	// 读取响应体
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	// 解析响应体中的 proto 数据
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

// 确保 httpGetter 实现了 PeerGetter 接口
var _ PeerGetter = (*httpGetter)(nil)
