package main

import (
	"flag"
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
	Debug      bool
	KcpMode    bool
	APIServer  string
}

var (
	comm     command
	auth     tr.IAuth
	connMgmt connStatManage
)

func main() {
	flag.StringVar(&comm.LocalPort, "local", ":10020", "server listen port")
	flag.StringVar(&comm.Redis, "redis", "", "redis address")
	flag.StringVar(&comm.Prometheus, "pms", "", "prometheus address")
	flag.BoolVar(&comm.KcpMode, "kcp", false, "listen on kcp mode")
	flag.BoolVar(&comm.Debug, "debug", false, "debug mode")
	flag.Parse()

	if comm.Debug == true {
		tr.Logger.Debug("debug mode")
		tr.EnableDebug()
	}
	if comm.Redis == "" {
		tr.Logger.Debug("authenticate user from environment")
		auth = tr.NewAuthFromEnv()
	} else {
		tr.Logger.Debug("authenticate user from redis")
		auth = tr.NewAuthFromRedis()
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
	block, _ := kcp.NewNoneBlockCrypt(nil)
	lis, err := kcp.ListenWithOptions(comm.LocalPort, block, 10, 3)
	if err != nil {
		tr.Logger.Fatal(err)
	}
	if err = lis.SetReadBuffer(4096 * 1024); err != nil {
		tr.Logger.Fatal(err)
	}
	if err = lis.SetWriteBuffer(4096 * 1024); err != nil {
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
		conn.SetNoDelay(0, 40, 2, 1)
		conn.SetMtu(1350)
		conn.SetWindowSize(1024, 1024)
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
	version      |         1            |   record version          0
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
	buf := make([]byte, 261)
	if _, err = conn.Read(buf[:35]); err != nil {
		return
	}
	switch buf[1] {
	case TCP_PROTO:
		token := string(buf[2:34])
		if _, ok := auth.Get(token); ok == false {
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
		err := tr.Copy(dst, conn)
		if err != nil {
			tr.Logger.Info(err)
		}
	}()
	err = tr.Copy(conn, dst)
	if err != nil {
		tr.Logger.Info(err)
	}
}
