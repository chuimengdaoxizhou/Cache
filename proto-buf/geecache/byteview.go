package geecache

// ByteView 表示字节数据的不可变视图
type ByteView struct {
	b []byte // 存储字节数据
}

// Len 返回视图的长度
func (v ByteView) Len() int {
	return len(v.b) // 返回字节数据的长度
}

// ByteSlice 返回字节数据的副本
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b) // 返回字节数据的副本
}

// String 返回字节数据的字符串表示，如果必要会进行复制
func (v ByteView) String() string {
	return string(v.b) // 将字节数据转换为字符串并返回
}

// cloneBytes 克隆字节数据，返回一个新的副本
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b)) // 创建一个新的字节切片
	copy(c, b)                // 将原字节切片的数据复制到新切片
	return c                  // 返回新的字节切片
}
