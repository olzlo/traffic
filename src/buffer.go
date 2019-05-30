package src

import (
	"errors"
	"io"
	"net"
	"sync"
)

var BufferPool = &bufferPool{
	p: make(map[int]*sync.Pool),
}

type bufferPool struct {
	p map[int]*sync.Pool
	m sync.RWMutex
}

func (b *bufferPool) Get(size int) []byte {
	if pool, ok := b.p[size]; ok {
		return pool.Get().([]byte)
	}
	b.m.Lock()
	defer b.m.Unlock()
	b.p[size] = &sync.Pool{
		New: func() interface{} {
			return make([]byte, size)
		},
	}
	return b.p[size].Get().([]byte)
}

func (b *bufferPool) Put(buf []byte) {
	if pool, ok := b.p[len(buf)]; ok {
		pool.Put(buf)
	} else {
		b.m.Lock()
		defer b.m.Unlock()
		b.p[len(buf)] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, len(buf))
			},
		}
		b.p[len(buf)].Put(buf)
	}
}

func CopyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if buf == nil || (buf != nil && len(buf) == 0) {
		err = errors.New("empty buf")
		return
	}
	for {
		if conn, ok := src.(net.Conn); ok {
			SetReadTimeout(conn)
		}
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return
}
