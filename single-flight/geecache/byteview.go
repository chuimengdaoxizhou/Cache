package geecache

// ByteView 表示对字节的不可变视图
type ByteView struct {
	b []byte // 存储字节数据
}

// Len 返回 ByteView 数据的长度
func (v ByteView) Len() int {
	return len(v.b) // 返回字节数据的长度
}

// ByteSlice 返回数据的副本，以字节切片的形式返回
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b) // 调用 cloneBytes 函数，返回字节数据的副本
}

// String 返回数据的字符串形式，如果需要会进行复制
func (v ByteView) String() string {
	return string(v.b) // 将字节数据转换为字符串并返回
}

// cloneBytes 创建并返回字节数据的副本
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b)) // 创建与原字节切片长度相同的新字节切片
	copy(c, b)                // 将原字节切片的数据复制到新字节切片中
	return c                  // 返回字节数据副本
}
