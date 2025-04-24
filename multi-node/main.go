package multi_node

import (
	"Cache/multi-node/geecache"
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

// 模拟一个数据库，键值对存储了人的名字和分数
var db = map[string]string{
	"Tom":  "630", // Tom 的分数是 630
	"Jack": "589", // Jack 的分数是 589
	"Sam":  "567", // Sam 的分数是 567
}

// createGroup 创建一个 geecache 的 Group，并指定它的缓存大小和 Getter
func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			// 模拟数据库查找数据
			if v, ok := db[key]; ok {
				return []byte(v), nil // 如果找到，返回对应的分数
			}
			// 如果没有找到，返回错误
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// startCacheServer 启动 geecache 缓存服务器，并将所有的 peers 注册到 Group 中
func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	peers := geecache.NewHTTPPool(addr) // 创建一个 HTTPPool 实例
	peers.Set(addrs...)                 // 设置其他 peer 节点
	gee.RegisterPeers(peers)            // 将 peers 注册到 Group
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers)) // 启动 HTTP 服务，监听缓存请求
}

// startAPIServer 启动 API 服务，允许外部通过 HTTP 请求获取缓存数据
func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key") // 获取请求中的 key 参数
			view, err := gee.Get(key)       // 从 geecache 中获取数据
			if err != nil {
				// 如果发生错误，返回 500 错误
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// 设置响应头，并返回缓存数据
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil)) // 启动 API 服务
}

func main() {
	var port int
	var api bool
	// 从命令行解析端口和是否启动 API 服务器的参数
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	// API 服务的地址
	apiAddr := "http://localhost:9999"
	// 节点地址映射
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	// 将所有节点地址添加到 addrs 列表中
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup() // 创建一个 geecache 的 Group
	// 如果需要启动 API 服务器，启动一个 Goroutine
	if api {
		go startAPIServer(apiAddr, gee)
	}
	// 启动缓存服务器
	startCacheServer(addrMap[port], addrs, gee)
}
