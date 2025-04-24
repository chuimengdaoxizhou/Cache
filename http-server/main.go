package http_server

/*
可以通过如下命令测试缓存服务：
$ curl http://localhost:9999/_geecache/scores/Tom
返回结果：630

$ curl http://localhost:9999/_geecache/scores/kkk
返回结果：kkk not exist
*/

import (
	"Cache/http-server/geecache"
	"fmt"
	"log"
	"net/http"
)

// 模拟一个数据库
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	// 创建一个名为 "scores" 的缓存组，最大缓存大小为 2KB，
	// 并定义当缓存未命中时的回调函数，从 db 中加载数据
	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	// 设置 HTTP 服务监听地址
	addr := "localhost:9999"
	// 初始化 HTTPPool，用于处理远程节点访问
	peers := geecache.NewHTTPPool(addr)
	log.Println("geecache is running at", addr)
	// 启动 HTTP 服务
	log.Fatal(http.ListenAndServe(addr, peers))
}
