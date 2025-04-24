package main

/*
$ curl "http://localhost:9999/api?key=Tom"
630

$ curl "http://localhost:9999/api?key=kkk"
kkk not exist
*/

import (
	"Cache/proto-buf/geecache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// createGroup 创建一个新的缓存组
func createGroup() *geecache.Group {
	// 创建一个名为 "scores" 的缓存组，缓存最大字节为 2<<10 (1024)，并指定 GetterFunc 函数
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key) // 模拟从数据库中查找键
			if v, ok := db[key]; ok {               // 如果数据库中有该键
				return []byte(v), nil // 返回值
			}
			return nil, fmt.Errorf("%s not exist", key) // 键不存在时返回错误
		}))
}

// startCacheServer 启动缓存服务器
func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	// 创建一个 HTTP 池，用于缓存的分布式访问
	peers := geecache.NewHTTPPool(addr)
	peers.Set(addrs...)      // 设置其他节点的地址
	gee.RegisterPeers(peers) // 注册缓存节点
	log.Println("geecache is running at", addr)
	// 启动 HTTP 服务，监听缓存请求
	log.Fatal(http.ListenAndServe(addr[7:], peers)) // 忽略前缀 "http://"
}

// startAPIServer 启动 API 服务器
func startAPIServer(apiAddr string, gee *geecache.Group) {
	// 处理 /api 路径的请求
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key") // 从 URL 查询参数中获取 key
			view, err := gee.Get(key)       // 从缓存中获取值
			if err != nil {                 // 如果出现错误
				http.Error(w, err.Error(), http.StatusInternalServerError) // 返回错误
				return
			}
			// 设置响应头为二进制流类型
			w.Header().Set("Content-Type", "application/octet-stream")
			// 写入缓存的值
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	// 启动 API 服务
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil)) // 忽略前缀 "http://"
}

func main() {
	// 定义命令行参数
	var port int
	var api bool
	// 设置端口和是否启动 API 服务器的标志
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999" // API 服务器地址
	// 定义可用的缓存节点地址
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	// 收集所有节点地址
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	// 创建一个缓存组
	gee := createGroup()

	// 如果设置了启动 API 服务器，则启动
	if api {
		go startAPIServer(apiAddr, gee)
	}

	// 启动缓存服务器
	startCacheServer(addrMap[port], addrs, gee)
}
