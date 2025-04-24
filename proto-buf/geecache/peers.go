package geecache

import pb "Cache/proto-buf/geecache/geecachepb"

// PeerPicker 是一个接口，必须实现该接口才能找到
// 拥有特定 key 的节点（peer）。
type PeerPicker interface {
	// PickPeer 根据 key 选择一个节点。
	// 返回节点和一个 bool 值，表示是否成功找到节点。
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 是一个接口，必须由节点实现，
// 用于从该节点获取数据。
type PeerGetter interface {
	// Get 向节点发送请求，获取指定数据。
	// 参数 `in` 是请求，`out` 是响应，返回错误信息（如果有）。
	Get(in *pb.Request, out *pb.Response) error
}
