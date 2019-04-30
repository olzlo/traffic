package main

import (
	"flag"
	"net"
	mo "traffic/src"
)

type Command struct {
	LocalPort  string
	Redis      string
	Prometheus string
	Debug      bool
}

var (
	comm Command
	auth mo.IAuth
)

func main() {
	flag.StringVar(&comm.LocalPort, "l", ":10020", "listen port")
	flag.StringVar(&comm.Redis, "rds", "", "redis address")
	flag.StringVar(&comm.Prometheus, "pms", "", "prometheus address")
	flag.BoolVar(&comm.Debug, "d", false, "debug mode")
	flag.Parse()

	if comm.Redis == "" {
		auth = mo.NewAuthWithEnv()
	} else {
		auth = mo.NewAuthWithRds()
	}
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
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {

}
