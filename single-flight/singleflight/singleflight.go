package singleflight

import "sync"

// call 用于表示正在进行中的请求
// 它包含一个 sync.WaitGroup 来等待请求的完成，
// 和一个 val 用于存储请求的结果，以及一个 err 用于存储请求过程中可能发生的错误
type call struct {
	wg  sync.WaitGroup // 用于等待请求完成
	val interface{}    // 存储请求的结果
	err error          // 存储请求的错误
}

// Group 代表一个请求组，防止同一个 key 被多次请求
// 它包含一个互斥锁（mu）和一个映射（m），
// 映射存储了每个 key 对应的请求信息
type Group struct {
	mu sync.Mutex       // 用于确保对 m 的并发访问是安全的
	m  map[string]*call // 存储正在进行中的请求
}

// Do 方法会根据传入的 key 和 fn 函数来保证相同的 key 只会执行一次 fn 函数
// 无论 Do 被调用多少次，fn 只会被调用一次，
// 其他的调用会等待 fn 执行完毕后返回相同的结果
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock() // 上锁，确保对 m 的安全访问
	// 如果 m 为空，初始化它
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 如果这个 key 对应的请求已经存在，直接等待并返回结果
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()       // 解锁，当前请求已在处理中
		c.wg.Wait()         // 等待请求完成
		return c.val, c.err // 返回请求的结果
	}

	// 如果 key 对应的请求不存在，创建一个新的 call
	c := new(call)
	c.wg.Add(1)   // 发起请求前，增加等待计数
	g.m[key] = c  // 将当前请求加入 m，表示该 key 对应的请求正在进行
	g.mu.Unlock() // 解锁，允许其他 goroutine 进行请求

	// 调用 fn 获取请求结果
	c.val, c.err = fn()
	c.wg.Done() // 请求完成，减少等待计数

	// 再次上锁，删除已完成请求的记录
	g.mu.Lock()
	delete(g.m, key) // 从 m 中删除该 key，表示请求已完成
	g.mu.Unlock()

	// 返回请求的结果或错误
	return c.val, c.err
}
