package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	tr "traffic/src"

	"github.com/aead/chacha20"
	kcp "github.com/xtaci/kcp-go"
)

type command struct {
	LocalPort, Token, ServerAddr, Key string
	KcpMode, Verbose                  bool
}

var (
	comm        command
	enKey       []byte
	connFactory func() (net.Conn, error)
)

func main() {
	flag.StringVar(&comm.LocalPort, "local", ":10020", "client listen port")
	flag.StringVar(&comm.ServerAddr, "server", "", "server address")
	flag.StringVar(&comm.Token, "token", "", "authenticate token")
	flag.StringVar(&comm.Key, "key", "", "shared key")
	flag.BoolVar(&comm.KcpMode, "kcp", false, "kcp mode")
	flag.BoolVar(&comm.Verbose, "verbose", false, "verbose mode")
	flag.Parse()

	if comm.ServerAddr == "" {
		tr.Logger.Fatal("server address must specified")
	}

	if comm.Token == "" {
		tr.Logger.Fatal("client must specify token")
	}

	if comm.Key == "" {
		tr.Logger.Fatal("client must specify key")
	} else {
		enKey = tr.EnforceKeys([]byte(comm.Key), 32)
	}
	if comm.Verbose {
		tr.EnableDebug()
	}
	if comm.KcpMode {
		connFactory = func() (net.Conn, error) {
			block, _ := kcp.NewNoneBlockCrypt(nil)
			conn, err := kcp.DialWithOptions(comm.ServerAddr, block, tr.DataShard, tr.ParityShard)
			if err != nil {
				return nil, err
			}
			conn.SetStreamMode(true)
			conn.SetWriteDelay(false)
			conn.SetNoDelay(tr.NoDelay, tr.Interval, tr.Resend, tr.NoCongestion)
			conn.SetWindowSize(tr.SndWnd, tr.RcvWnd)
			conn.SetMtu(tr.MTU)
			conn.SetACKNoDelay(true)

			if err := conn.SetDSCP(tr.DSCP); err != nil {
				return nil, err
			}
			if err := conn.SetReadBuffer(tr.SockBuf); err != nil {
				return nil, err
			}
			if err := conn.SetWriteBuffer(tr.SockBuf); err != nil {
				return nil, err
			}
			return tr.NewEncryptConn(conn, enKey, chacha20.NewCipher), nil
		}
	} else {
		connFactory = func() (conn net.Conn, err error) {
			if conn, err = net.Dial("tcp", comm.ServerAddr); err != nil {
				return
			}
			return tr.NewEncryptConn(conn, enKey, chacha20.NewCipher), nil
		}
	}

	tr.Logger.Info("sock5 server listen at :", comm.LocalPort)
	tcpListen()
}

func tcpListen() {
	lis, err := net.Listen("tcp", ":"+comm.LocalPort)
	if err != nil {
		tr.Logger.Fatal(err)
	}
	for {
		conn, err := lis.Accept()
		if err != nil {
			tr.Logger.Fatal(err)
		}
		tr.Logger.Debug("new client from ", conn.RemoteAddr())
		go handleRequest(conn)
	}
}

func handShake(conn net.Conn) (err error) {
	buf := make([]byte, 255)
	//sock5 defined by rfc1928
	if _, err = io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	if buf[0] != 5 {
		return errors.New("sock version mismatch")
	}
	nmethod := int(buf[1])
	if nmethod > 0 {
		// has more methods to read, rare case
		if _, err = io.ReadFull(conn, buf[:nmethod]); err != nil {
			return
		}
	}
	// send confirmation: version 5, no authentication required
	_, err = conn.Write([]byte{5, 0})
	if err != nil {
		tr.Logger.Error(err)
		return
	}
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
	var buf [262]byte
	if _, err = io.ReadFull(conn, buf[:5]); err != nil {
		return
	}
	if buf[0] != 5 {
		err = fmt.Errorf("client socks version %d mismatch", buf[0])
		return
	}
	if buf[1] != 1 {
		err = errors.New("client cmd not support")
		return
	}
	switch buf[3] {
	case 1:
		if _, err = io.ReadFull(conn, buf[5:10]); err != nil {
			return
		}
		addr := &net.TCPAddr{
			IP:   net.IPv4(buf[4], buf[5], buf[6], buf[7]),
			Port: int(binary.BigEndian.Uint16(buf[8:10])),
		}
		host = addr.String()
	case 3:
		dmlen := buf[4]
		if _, err = io.ReadFull(conn, buf[5:dmlen+7]); err != nil {
			return
		}
		port := int(binary.BigEndian.Uint16(buf[dmlen+5 : dmlen+7]))
		host = fmt.Sprintf("%s:%d", string(buf[5:dmlen+5]), port)
	default:
		err = errors.New("address type not support")
		return
	}
	return
}

func createServerConn(host string) (conn net.Conn, err error) {
	if conn, err = connFactory(); err != nil {
		return
	}
	var buf [512]byte
	//ver
	buf[0] = 1
	//proto
	buf[1] = tr.TCP_PROTO
	//token length
	buf[2] = byte(len(comm.Token))
	copy(buf[3:35], comm.Token)
	buf[35] = byte(len(host))
	copy(buf[36:], host)

	_, err = conn.Write(buf[:len(host)+36])
	return
}

func handleRequest(c net.Conn) {
	defer c.Close()
	err := handShake(c)
	if err != nil {
		tr.Logger.Error(err)
		return
	}
	host, err := getRequest(c)
	if err != nil {
		tr.Logger.Error(err)
		return
	}
	_, err = c.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
	if err != nil {
		tr.Logger.Error(err)
		return
	}
	s, err := createServerConn(host)
	if err != nil {
		tr.Logger.Error(err)
		return
	}
	_, _, err = tr.Pipe(c, s)
	if err != nil {
		tr.Logger.Error(err)
	}
}
