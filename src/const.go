package src

const (
	TCP_PROTO = iota
	UDP_PROTO
)

//kcp const
const (
	DataShard, ParityShard = 10, 3
	//normal
	NoDelay, Interval, Resend, NoCongestion = 0, 30, 2, 1
	SndWnd, RcvWnd                          = 1024, 1024
	MTU                                     = 1350
	SockBuf                                 = 4194304
	DSCP                                    = 0
)
