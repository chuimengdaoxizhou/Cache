package single_flight

import (
	"Cache/single-flight/geecache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

/*
   $ curl "http://localhost:9999/api?key=Tom"
   630

   $ curl "http://localhost:9999/api?key=kkk"
   kkk not exist
*/

// 模拟数据库，用于存储一些简单的键值对
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// 创建一个新的 geecache.Group，模拟从数据库中查询
func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			// 如果数据库中没有该 key，返回错误
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// 启动缓存服务器，用于处理来自其他节点的请求
func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	// 创建 HTTP 池，设置缓存服务器的地址
	peers := geecache.NewHTTPPool(addr)
	peers.Set(addrs...)      // 将所有节点的地址加入到缓存池中
	gee.RegisterPeers(peers) // 将 peers 注册到 geecache 中
	log.Println("geecache is running at", addr)
	// 启动服务器，监听传入的请求
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 启动 API 服务器，用于接收前端请求并返回缓存的结果
func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// 获取请求中的 key 参数
			key := r.URL.Query().Get("key")
			// 从 geecache 中获取对应的值
			view, err := gee.Get(key)
			if err != nil {
				// 如果出错，返回错误信息
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// 设置返回的内容类型为二进制流，并返回结果
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	// 启动 API 服务器，监听请求
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	// 定义命令行参数，用于设置端口和是否启动 API 服务器
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	// 设置 API 服务器的地址
	apiAddr := "http://localhost:9999"
	// 设置缓存服务器的地址映射
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	// 将地址映射中的所有地址存储到 addrs 切片中
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	// 创建 geecache 实例
	gee := createGroup()
	// 如果启用 API 服务器，则启动它
	if api {
		go startAPIServer(apiAddr, gee)
	}
	// 启动缓存服务器
	startCacheServer(addrMap[port], addrs, gee)
}
