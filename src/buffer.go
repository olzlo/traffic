package src

import (
	"sync"
)

const bufSize = 4096

var bufPool = &bufferPool{
	p: sync.Pool{
		New: func() interface{} {
			return make([]byte, bufSize)
		},
	},
}

type bufferPool struct {
	p sync.Pool
}

func (b *bufferPool) Get() []byte {
	return b.p.Get().([]byte)
}

func (b *bufferPool) Put(buf []byte) {
	if len(buf) == bufSize {
		b.p.Put(buf)
	} else {
		panic("put different size buffer")
	}
}
