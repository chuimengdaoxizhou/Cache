package geecache

// PeerPicker 是一个接口，必须实现该接口才能定位到
// 拥有特定 key 的 peer 节点。
type PeerPicker interface {
	// PickPeer 根据 key 选择一个 peer 节点
	// 返回值 peer 是实现了 PeerGetter 接口的对象，ok 表示是否找到了该 peer
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 是一个接口，必须由 peer 节点实现
// 该接口提供了从远程节点获取数据的方法。
type PeerGetter interface {
	// Get 从指定的 group 中根据 key 获取数据
	// 返回字节数据和可能的错误
	Get(group string, key string) ([]byte, error)
}
