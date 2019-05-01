package main

import (
	"flag"
	"github.com/sirupsen/logrus"
	"net"
	"strings"
	"sync"
	tr "traffic/src"
)

type command struct {
	LocalPort  string
	Redis      string
	Prometheus string
	Debug      bool
}

var (
	comm    command
	auth    tr.IAuth
	cmanage connStatManage
)

func main() {
	flag.StringVar(&comm.LocalPort, "l", ":10020", "listen port")
	flag.StringVar(&comm.Redis, "rds", "", "redis address")
	flag.StringVar(&comm.Prometheus, "pms", "", "prometheus address")
	flag.BoolVar(&comm.Debug, "d", false, "debug mode")
	flag.Parse()

	if comm.Debug == false {
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

	tr.Logger.WithFields(logrus.Fields{
		"port": comm.LocalPort,
	}).Debug("tcp listen")
	tcpListen()
}

func tcpListen() {
	ln, err := net.Listen("tcp", comm.LocalPort)
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(tr.NewConn(conn,&tr.Cipher{}))
	}
}

type connStatManage struct {
	m         sync.Mutex
	connected map[string]int
}

func (cm *connStatManage) isConnected(conn net.Conn) bool {
	tc := conn.(*net.TCPConn)
	addr := tc.RemoteAddr().String()
	addr = addr[:strings.Index(addr, ":")]
	cm.m.Lock()
	defer cm.m.Unlock()
	if _, ok := cm.connected[addr]; ok {
		cm.connected[addr]++
		return true
	}
	cm.connected[addr] = 1
	return false
}

func handshake(conn *tr.Conn) bool {

	return false
}

func handleConnection(conn *tr.Conn) {
	defer conn.Close()
	if cmanage.isConnected(conn) == false && handshake(conn) == false {
		return
	}

}
