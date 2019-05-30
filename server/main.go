package main

import (
	"flag"
	"fmt"
	"net"
	tr "traffic/src"

	"errors"

	"github.com/aead/chacha20"
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
		tr.Logger.Debug("tcp listen port ", comm.LocalPort)
		tcpListen()
	} else {
		tr.Logger.Debug("kcp listen port ", comm.LocalPort)
		kcpListen()
	}
}

func kcpListen() {
	block, _ := kcp.NewNoneBlockCrypt(nil)
	lis, err := kcp.ListenWithOptions(comm.LocalPort, block, tr.DataShard, tr.ParityShard)
	if err != nil {
		tr.Logger.Fatal(err)
	}
	if err = lis.SetReadBuffer(tr.SockBuf); err != nil {
		tr.Logger.Fatal(err)
	}
	if err = lis.SetWriteBuffer(tr.SockBuf); err != nil {
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
		conn.SetNoDelay(tr.NoDelay, tr.Interval, tr.Resend, tr.NoCongestion)
		conn.SetMtu(tr.MTU)
		conn.SetWindowSize(tr.SndWnd, tr.RcvWnd)
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
		tr.Logger.Debug("remote client accept ", conn.RemoteAddr())
		go handleConnection(tr.NewEncryptConn(conn, auth.SharedKey(), chacha20.NewCipher))
	}
}

/*
    field                bytes                 description         index
    ------------------------------------------------------------------------
	version      |         1            |   version              |   0
	protocol     |         1            |   layer-4 protocol     |   1
	token_len    |         1            |   token len            |   2
	token        |         32           |   user distinguish     |   3
	dst_len      |         1            |   dst addr len         |   35
	dst_address  |  no more than 261    |   form as addr:port
*/

func authenticate(conn *tr.Conn) (addr net.Addr, err error) {
	buf := tr.BufferPool.Get(261)
	defer tr.BufferPool.Put(buf)
	tr.SetReadTimeout(conn)
	if _, err = conn.Read(buf[:36]); err != nil {
		return
	}
	if buf[0] != 1 {
		return nil, fmt.Errorf("client ver: %d not match with server ver 1", buf[0])
	}

	switch buf[1] {
	case tr.TCP_PROTO:
		tLen := int(buf[2])
		if tLen > 32 {
			return nil, fmt.Errorf("token length %d overflow", tLen)
		}
		token := string(buf[3 : 3+tLen])
		if ok := auth.IsValid(token); !ok {
			return nil, fmt.Errorf("unauthorized user token %s", token)
		}
		conn.Token = token
		dstLen := int(buf[35])
		if dstLen > 261 {
			return nil, errors.New("invalid length of address")
		}
		if _, err = conn.Read(buf[:dstLen]); err != nil {
			return
		}
		if addr, err = net.ResolveTCPAddr("tcp", string(buf[:dstLen])); err != nil {
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
	tr.Logger.Debugf("user:%s src:%s dst %s", conn.Token, conn.RemoteAddr(), addr.String())
	dst, err = net.Dial(addr.Network(), addr.String())
	return
}

func handleConnection(conn *tr.Conn) {
	defer conn.Close()
	dst, err := getdstConn(conn)
	if err != nil {
		tr.Logger.Error(err)
		return
	}
	defer dst.Close()
	go func() {
		if err := tr.Copy(dst, conn); err != nil {
			tr.Logger.Info("client-->server  ", err)
		}
	}()
	err = tr.Copy(conn, dst)
	if err != nil {
		tr.Logger.Info("server<--client  ", err)
	}
}
