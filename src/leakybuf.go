package src

type LeakyBuf struct {
	size int
	list chan []byte
}

var leakyBuf = NewLeakyBuf(4096, 2048)

func NewLeakyBuf(bsize, lsize int) *LeakyBuf {
	return &LeakyBuf{
		bsize,
		make(chan []byte, lsize),
	}
}

func (l *LeakyBuf) Get() (buf []byte) {
	select {
	case buf = <-l.list:
	default:
		buf = make([]byte, l.size)
	}
	return
}

func (l *LeakyBuf) Put(buf []byte) {
	if len(buf) != l.size {
		panic("buffer size mismatch")
	}
	select {
	case l.list <- buf:
	default:
	}
}
