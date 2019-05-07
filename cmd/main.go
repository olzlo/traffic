package main

import (
	"flag"
	"net"
	"sync"
	tr "traffic/src"

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
}

var (
	comm     command
	auth     tr.IAuth
	connMgmt connStatManage
)

func main() {
	flag.StringVar(&comm.LocalPort, "l", ":10020", "listen port")
	flag.StringVar(&comm.Redis, "redis", "", "redis address")
	flag.StringVar(&comm.Prometheus, "pms", "", "prometheus address")
	flag.BoolVar(&comm.KcpMode, "kcp", false, "listen on kcp mode")
	flag.BoolVar(&comm.Debug, "d", false, "debug mode")
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
		tcpListen(auth)
	} else {
		tr.Logger.WithFields(logrus.Fields{
			"port": comm.LocalPort,
		}).Debug("kcp listen")
		kcpListen(auth)
	}
}

func kcpListen(auth tr.IAuth) {
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

func tcpListen(auth tr.IAuth) {
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

type connStatManage struct {
	m         sync.Mutex
	connected map[string]int
}

func (cm *connStatManage) AddCounter(conn *tr.Conn) (err error) {
	addr := conn.RemoteIP()
	cm.m.Lock()
	defer cm.m.Unlock()
	if _, ok := cm.connected[addr]; ok {
		cm.connected[addr]++
		return
	}
	if err = cryptoUpdate(conn); err == nil {
		cm.connected[addr] = 1
	}
	return
}

func (cm *connStatManage) DownCounter(conn *tr.Conn) {
	addr := conn.RemoteIP()
	cm.m.Lock()
	defer cm.m.Unlock()
	if _, ok := cm.connected[addr]; ok {
		cm.connected[addr]--
	}
}

type pkgType int

const (
	MetaPkg pkgType = iota
	DataPkg
)

/*  MetaPkg header info
+---------+-----------+-----------+----------------------+---------------+-----------+----------+
| version |  pkg_type | sock_type |  data_len (udp only) | dst_addr_type |  dst_addr | dst_port |
+---------+-----------+-----------+----------------------+---------------+-----------+----------+
|    1    |      1    |    1      |          2           |       1       |    val    |    2     |
+---------+-----------+-----------+----------------------+---------------+-----------+----------+
|  uid    |
+---------+
|  val    |
+---------+
*/

func cryptoUpdate(conn *tr.Conn) error {
	/* step 1: authentication
	   step 2: key negotiation
	   step 3: connection establish
	*/

	return nil
}

func getdstConn(conn *tr.Conn) (dst net.Conn, err error) {

	return
}

func handleConnection(conn *tr.Conn) {
	defer conn.Close()
	if err := connMgmt.AddCounter(conn); err != nil {
		tr.Logger.Error(err)
		return
	}
	defer connMgmt.DownCounter(conn)
	dst, err := getdstConn(conn)
	if err != nil {
		tr.Logger.Info(err)
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
