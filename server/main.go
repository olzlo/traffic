package main

import (
	"flag"
	"fmt"
	"net"
	tr "traffic/src"

	"errors"

	"github.com/aead/chacha20"
	"github.com/sirupsen/logrus"
	kcp "github.com/xtaci/kcp-go"
)

type command struct {
	LocalPort  string
	Redis      string
	Prometheus string
	Verbose    bool
	KcpMode    bool
	APIServer  string
}

var (
	comm command
	auth tr.IAuth
)

func main() {
	flag.StringVar(&comm.LocalPort, "local", ":10020", "server listen port")
	flag.StringVar(&comm.Redis, "redis", "", "redis address")
	flag.StringVar(&comm.Prometheus, "pms", "", "prometheus address")
	flag.BoolVar(&comm.KcpMode, "kcp", false, "listen on kcp mode")
	flag.BoolVar(&comm.Verbose, "verbose", false, "verbose mode")
	flag.Parse()

	if comm.Verbose == true {
		tr.EnableDebug()
	}
	if comm.Redis == "" {
		tr.Logger.Debug("authenticate user from environment")
		auth = tr.NewAuthFromEnv()
	} else {
		tr.Logger.Debug("authenticate user from redis")
		auth = tr.NewAuthFromRedis(comm.Redis)
	}

	if comm.KcpMode == false {
		tr.Logger.WithFields(logrus.Fields{
			"port": comm.LocalPort,
		}).Debug("tcp listen")
		tcpListen()
	} else {
		tr.Logger.WithFields(logrus.Fields{
			"port": comm.LocalPort,
		}).Debug("kcp listen")
		kcpListen()
	}
}

func kcpListen() {
	const (
		DataShard, ParityShard = 10, 3
		//normal
		NoDelay, Interval, Resend, NoCongestion = 0, 30, 2, 1
		SndWnd, RcvWnd                          = 1024, 1024
		MTU                                     = 1350
		SockBuf                                 = 4194304
		DSCP                                    = 0
	)

	block, _ := kcp.NewNoneBlockCrypt(nil)
	lis, err := kcp.ListenWithOptions(comm.LocalPort, block, DataShard, ParityShard)
	if err != nil {
		tr.Logger.Fatal(err)
	}
	if err = lis.SetReadBuffer(SockBuf); err != nil {
		tr.Logger.Fatal(err)
	}
	if err = lis.SetWriteBuffer(SockBuf); err != nil {
		tr.Logger.Fatal(err)
	}
	for {
		conn, err := lis.AcceptKCP()
		if err != nil {
			tr.Logger.Fatal(err)
		}

		conn.SetStreamMode(true)
		conn.SetWriteDelay(false)
		//fast
		conn.SetNoDelay(NoDelay, Interval, Resend, NoCongestion)
		conn.SetMtu(MTU)
		conn.SetWindowSize(SndWnd, RcvWnd)
		conn.SetACKNoDelay(true)
		go handleConnection(tr.NewEncryptConn(conn, auth.SharedKey(), chacha20.NewCipher))
	}
}

func tcpListen() {
	lis, err := net.Listen("tcp", comm.LocalPort)
	if err != nil {
		tr.Logger.Fatal(err)
	}
	for {
		conn, err := lis.Accept()
		if err != nil {
			tr.Logger.Fatal(err)
		}
		go handleConnection(tr.NewEncryptConn(conn, auth.SharedKey(), chacha20.NewCipher))
	}
}

/*
    field                bytes                 description
    ------------------------------------------------------------
	version      |         1            |   version                 0
	protocol     |         1            |   layer-4 protocol        1
	token        |         32           |   user distinguish        2
	dst_len      |         1            |   dst addr len            34
	dst_address  |  no more than 261    |   form as addr:port
*/

const (
	TCP_PROTO = iota
	UDP_PROTO
)

func authenticate(conn *tr.Conn) (addr net.Addr, err error) {
	buf := tr.BufferPool.Get(261)
	defer tr.BufferPool.Put(buf)
	tr.SetReadTimeout(conn)
	if _, err = conn.Read(buf[:35]); err != nil {
		return
	}
	if buf[0] != 1 {
		return nil, fmt.Errorf("client ver: %d not match with server ver 1", buf[0])
	}

	switch buf[1] {
	case TCP_PROTO:
		token := string(buf[2:34])
		if ok := auth.IsValid(token); ok == false {
			return nil, errors.New("unauthorized user token")
		}
		l := int(buf[34])
		if l > 261 {
			return nil, errors.New("invalid length of address")
		}

		if _, err = conn.Read(buf[:l]); err != nil {
			return
		}
		if addr, err = net.ResolveTCPAddr("tcp", string(buf[:l])); err != nil {
			return
		}
	default:
		err = errors.New("unsupported protocol type")
	}
	return
}

func getdstConn(conn *tr.Conn) (dst net.Conn, err error) {
	addr, err := authenticate(conn)
	if err != nil {
		return
	}
	dst, err = net.Dial(addr.Network(), addr.String())
	return
}

func handleConnection(conn *tr.Conn) {
	defer conn.Close()
	dst, err := getdstConn(conn)
	if err != nil {
		tr.Logger.Error(err)
	}
	defer dst.Close()
	go func() {
		if err := tr.Copy(dst, conn); err != nil {
			tr.Logger.Info(err)
		}
	}()
	err = tr.Copy(conn, dst)
	if err != nil {
		tr.Logger.Info(err)
	}
}