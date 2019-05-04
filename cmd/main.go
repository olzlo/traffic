package main

import (
	"flag"
	"net"
	"sync"
	tr "traffic/src"

	"github.com/sirupsen/logrus"
)

type command struct {
	LocalPort  string
	Redis      string
	Prometheus string
	Debug      bool
}

var (
	comm      command
	auth      tr.IAuth
	cmanage   connStatManage
	sharedkey []byte
)

func main() {
	flag.StringVar(&comm.LocalPort, "l", ":10020", "listen port")
	flag.StringVar(&comm.Redis, "rds", "", "redis address")
	flag.StringVar(&comm.Prometheus, "pms", "", "prometheus address")
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
		auth = tr.NewAuthFromRds()
	}

	if len(comm.Prometheus) > 0 {
		tr.EnableMetrics(comm.Prometheus)
	}

	var ok bool
	if sharedkey, ok = auth.SharedKey(); ok {
		tr.Logger.Fatal("must set shared key")
	}

	tr.Logger.WithFields(logrus.Fields{
		"port": comm.LocalPort,
	}).Debug("tcp listen")
	tcpListen()
}

func tcpListen() {
	ln, err := net.Listen("tcp", comm.LocalPort)
	if err != nil {
		tr.Logger.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			tr.Logger.Fatal(err)
		}
		go handleConnection(tr.NewConn(conn, &tr.Cipher{
			Key: sharedkey,
		}))
	}
}

type connStatManage struct {
	m         sync.Mutex
	connected map[string]int
}

func (cm *connStatManage) isConnected(conn *tr.Conn) bool {
	addr := conn.RemoteIP()
	cm.m.Lock()
	defer cm.m.Unlock()
	if _, ok := cm.connected[addr]; ok {
		cm.connected[addr]++
		return true
	}
	if cryptoUpdate(conn) {
		cm.connected[addr] = 1
	}
	return false
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
*/

func cryptoUpdate(conn *tr.Conn) bool {

	return false
}

func getdstConn(conn *tr.Conn) (dst net.Conn, err error) {

	return
}

func handleConnection(conn *tr.Conn) {
	defer conn.Close()
	if cmanage.isConnected(conn) == false {
		return
	}
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
