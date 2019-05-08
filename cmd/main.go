package main

import (
	"flag"
	"fmt"
	"net"
	"sync"
	tr "traffic/src"

	"github.com/aead/chacha20"
	"github.com/sirupsen/logrus"
	kcp "github.com/xtaci/kcp-go"
)

type command struct {
	LocalPort  string
	HandPort   string
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
	flag.StringVar(&comm.LocalPort, "local", ":10020", "server listen port")
	flag.StringVar(&comm.HandPort, "hand", ":10021", "handshake listen port")
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

	go handShakeListen(auth)

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
		if stat, ok := connMgmt.GetConnStat(conn); ok {
			if key, ok := auth.GetKey(stat.user); ok {
				conn.SetStreamMode(true)
				conn.SetWriteDelay(false)
				//fast
				conn.SetNoDelay(0, 40, 2, 1)
				conn.SetMtu(1350)
				conn.SetWindowSize(1024, 1024)
				conn.SetACKNoDelay(true)
				go handleConnection(tr.NewEncryptConn(conn, key, chacha20.NewCipher))
			} else {
				tr.Logger.Info(fmt.Sprintf("user %s not exist", stat.user))
				conn.Close()
			}
		} else {
			tr.Logger.Info("unauthorized connection from ", tr.RemoteIP(conn))
			conn.Close()
		}
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
		if stat, ok := connMgmt.GetConnStat(conn); ok {
			if key, ok := auth.GetKey(stat.user); ok {
				go handleConnection(tr.NewEncryptConn(conn, key, chacha20.NewCipher))
			} else {
				tr.Logger.Info(fmt.Sprintf("user %s not exist", stat.user))
				conn.Close()
			}
		} else {
			tr.Logger.Info("unauthorized connection from ", tr.RemoteIP(conn))
			conn.Close()
		}
	}
}

type connStat struct {
	count int
	user  string
}

type connStatManage struct {
	m         sync.RWMutex
	connected map[string]*connStat
}

func (cm *connStatManage) Bind(conn net.Conn) (add func(), down func()) {
	addr := tr.RemoteIP(conn)
	add = func() {
		cm.m.Lock()
		defer cm.m.Unlock()
		if _, ok := cm.connected[addr]; ok {
			cm.connected[addr].count++
		}
	}
	down = func() {
		cm.m.Lock()
		defer cm.m.Unlock()
		if _, ok := cm.connected[addr]; ok {
			cm.connected[addr].count--
		}
	}
	return
}

func (cm *connStatManage) GetConnStat(conn net.Conn) (stat *connStat, ok bool) {
	cm.m.RLock()
	defer cm.m.RUnlock()
	stat, ok = cm.connected[tr.RemoteIP(conn)]
	return
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
| uid_len |    uid    |
+---------+-----------+
|   1     |	   val    |
+---------+-----------+
*/

func getdstConn(conn *tr.Conn) (dst net.Conn, err error) {

	return
}

func handShakeListen(auth tr.IAuth) {
	lis, err := net.Listen("tcp", comm.HandPort)
	if err != nil {
		tr.Logger.Fatal(err)
	}
	for {
		conn, err := lis.Accept()
		if err != nil {
			tr.Logger.Fatal(err)
		}
		go handleHandShake(tr.NewEncryptConn(conn, auth.SharedKey(), chacha20.NewCipher))
	}
}

func handleHandShake(conn *tr.Conn) {

}

func handleConnection(conn *tr.Conn) {
	defer conn.Close()
	add, done := connMgmt.Bind(conn.Conn)
	add()
	defer done()
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
