package consistent_hash

/*
$ curl http://localhost:9999/_geecache/scores/Tom
630

$ curl http://localhost:9999/_geecache/scores/kkk
kkk not exist
*/

import (
	"Cache/consistent-hash/geecache"
	"fmt"
	"log"
	"net/http"
)

// 模拟数据库，存储一些键值对数据
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	// 创建一个名为 "scores" 的缓存组，缓存容量为 2^10 字节
	// 当缓存中没有目标 key 时，通过 GetterFunc 从数据库加载数据
	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key) // 模拟从数据库读取数据
			if v, ok := db[key]; ok {
				return []byte(v), nil // 数据存在，返回数据
			}
			return nil, fmt.Errorf("%s not exist", key) // 数据不存在，返回错误
		}))

	// 设置 HTTP 服务监听地址
	addr := "localhost:9999"
	// 创建一个 HTTP 节点池，用于处理节点间的缓存请求
	peers := geecache.NewHTTPPool(addr)
	log.Println("geecache is running at", addr) // 打印服务启动信息
	// 启动 HTTP 服务，监听并处理来自客户端的请求
	log.Fatal(http.ListenAndServe(addr, peers))
}
