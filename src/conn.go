package src

import (
	"net"
)

type Conn struct {
	net.Conn
	*Cipher
}

func NewConn(c net.Conn, p *Cipher) *Conn {
	return &Conn{
		c, p,
	}
}

func (c *Conn) Write(b []byte) (n int, err error) {

	return
}

func (c *Conn) Read(b []byte) (n int, err error) {

	return
}
