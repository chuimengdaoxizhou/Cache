package geecache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// 提供被其他节点通过 HTTP 访问缓存的能力（即 HTTP 服务端）

// 默认的访问路径前缀，例如：http://localhost:8001/_geecache/
const defaultBasePath = "/_geecache/"

// HTTPPool 代表一个 HTTP 服务端节点，处理来自其他节点的请求
type HTTPPool struct {
	self     string // 记录自己的地址，例如 "http://localhost:8001"
	basePath string // 节点间通信地址的前缀，默认是 /_geecache/
}

// NewHTTPPool 初始化一个 HTTP 节点池，并设置自身地址
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log 打印日志，带上节点身份前缀
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 是 HTTPPool 实现 http.Handler 接口的方法，处理所有以 basePath 开头的请求
// 请求格式：/_geecache/<groupname>/<key>
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 校验请求路径是否以设定的 basePath 开头
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	// 打印访问日志
	p.Log("%s %s", r.Method, r.URL.Path)

	// 提取 group 和 key，例如 /_geecache/scores/Tom => groupName=scores, key=Tom
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// 根据 group 名称查找对应的 Group 实例
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 从 Group 中获取缓存数据
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回数据（二进制形式）
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}
