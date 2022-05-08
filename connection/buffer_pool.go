package connection

import "sync"

const (
	defaultBufSize = 4096
)

var bufferPool sync.Pool

func GetBuffer() []byte {
	return bufferPool.Get().([]byte)
}

func PutBuffer(b []byte) {
	bufferPool.Put(b)
}

func init() {
	bufferPool.New = func() interface{} {
		return make([]byte, defaultBufSize)
	}
}
