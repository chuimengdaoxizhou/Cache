package geecache

// ByteView 结构体表示字节的不可变视图。
type ByteView struct {
	b []byte // 存储字节数据
}

// Len 方法返回 ByteView 的字节长度。
func (v ByteView) Len() int {
	return len(v.b) // 返回字节数组的长度
}

// ByteSlice 方法返回数据的副本，类型为字节切片（[]byte）。
// 通过调用 cloneBytes 函数来确保返回的是数据的副本，而不是原始数据的引用。
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b) // 返回数据的副本
}

// String 方法返回数据的字符串形式。
// 如果必要的话，会创建数据的副本。
func (v ByteView) String() string {
	return string(v.b) // 将字节数据转换为字符串
}

// cloneBytes 函数用于复制字节切片，确保返回的数据副本与原数据独立。
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b)) // 创建一个与原字节切片长度相同的空切片
	copy(c, b)                // 将原字节切片的内容复制到新切片中
	return c                  // 返回新切片（副本）
}
