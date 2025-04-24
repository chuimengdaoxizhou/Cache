package single_node

// ByteView 是缓存值的只读视图（immutable view）。
// b 存储真实的缓存值。选择 []byte 类型是因为它可以表示任意类型的数据，
// 如字符串、图片、序列化对象等，是一种通用的数据表示方式。
type ByteView struct {
	b []byte
}

// Len 返回缓存值的长度（单位：字节）
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice 返回缓存数据的副本，避免外部修改原始数据（确保只读）
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// String 将缓存数据转换为字符串（常用于字符串类型的缓存值）
func (v ByteView) String() string {
	return string(v.b)
}

// cloneBytes 拷贝一份新的 byte 切片，保证数据隔离
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b)) // 创建一个新的切片
	copy(c, b)                // 拷贝数据到新切片
	return c
}
