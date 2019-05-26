package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	tr "traffic/src"
)

type command struct {
	LocalPort, Token string
	KcpMode, Verbose bool
}

var (
	comm command
)

func main() {
	flag.StringVar(&comm.LocalPort, "local", ":10020", "client listen port")
	flag.StringVar(&comm.Token, "token", "", "authenticate token")
	flag.BoolVar(&comm.KcpMode, "kcp", false, "kcp mode")
	flag.BoolVar(&comm.Verbose, "verbose", false, "verbose mode")
	flag.Parse()

	if comm.Token == "" {
		tr.Logger.Fatal("client must specify token")
	}

	if comm.Verbose == true {
		tr.EnableDebug()
	}
	tr.Logger.Info("sock5 server listen at :", comm.LocalPort)
	tcpListen()
}

func tcpListen() {
	lis, err := net.Listen("tcp", ":"+comm.LocalPort)
	if err != nil {
		tr.Logger.Fatal(err)
	}
	conn, err := lis.Accept()
	if err != nil {
		tr.Logger.Fatal(err)
	}
	go handleRequest(conn)
}

func handShake(conn net.Conn) (err error) {
	buf := make([]byte, 258)
	tr.SetReadTimeout(conn)
	//sock5 defined by rfc1928
	if _, err = io.ReadAtLeast(conn, buf, 2); err != nil {
		return
	}
	if buf[0] != 5 {
		return errors.New("sock version mismatch")
	}
	nmethod := int(buf[1])
	if nmethod > 0 { // has more methods to read, rare case
		if _, err = io.ReadFull(conn, buf[2:2+nmethod]); err != nil {
			return
		}
	}
	// send confirmation: version 5, no authentication required
	_, err = conn.Write([]byte{5, 0})
	return
}

/*
		  The SOCKS request is formed as follows:

	        +----+-----+-------+------+----------+----------+
	        |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	        +----+-----+-------+------+----------+----------+
	        | 1  |  1  | X'00' |  1   | Variable |    2     |
	        +----+-----+-------+------+----------+----------+

	     Where:

	          o  VER    protocol version: X'05'
	          o  CMD
	             o  CONNECT X'01'
	             o  BIND X'02'
	             o  UDP ASSOCIATE X'03'
	          o  RSV    RESERVED
	          o  ATYP   address type of following address
	             o  IP V4 address: X'01'
	             o  DOMAINNAME: X'03'
	             o  IP V6 address: X'04'
	          o  DST.ADDR       desired destination address
	          o  DST.PORT desired destination port in network octet
	             order
*/

func getRequest(conn net.Conn) (host string, err error) {
	buf := tr.BufferPool.Get(262)
	if buf[0] != 5 {
		err = errors.New("client socks version mismatch")
		return
	}
	if buf[1] != 1 {
		err = errors.New("client cmd not support")
		return
	}
	switch buf[3] {
	case 1:
		addr := &net.TCPAddr{
			IP:   net.IPv4(buf[4], buf[5], buf[6], buf[7]),
			Port: int(binary.BigEndian.Uint16(buf[8:10])),
		}
		host = addr.String()
	case 3:
		dmlen := buf[4]
		port := int(binary.BigEndian.Uint16(buf[dmlen+5 : dmlen+5+2]))
		host = fmt.Sprintf("%s:%d", string(buf[5:dmlen+5]), port)
	default:
		err = errors.New("address type not support")
		return
	}
	return
}

func createServerConn(host string) (conn net.Conn, err error) {

	return
}

func handleRequest(c net.Conn) {
	defer c.Close()
	err := handShake(c)
	if err != nil {
		tr.Logger.Info(err)
	}
	host, err := getRequest(c)
	if err != nil {
		tr.Logger.Info(err)
	}
	var res [10]byte
	//ver
	res[1] = 5
	//atype
	res[4] = 1
	if _, err = c.Write(res[:]); err != nil {
		tr.Logger.Fatal(err)
	}
	s, err := createServerConn(host)
	if err != nil {
		tr.Logger.Info(err)
	}
	defer s.Close()
	go func() {
		if err := tr.Copy(c, s); err != nil {
			tr.Logger.Info(err)
		}
	}()

	err = tr.Copy(s, c)
	if err != nil {
		tr.Logger.Info(err)
	}

}
