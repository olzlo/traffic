package src

import (
	"io"
	"net"
	"time"
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

func Copy(dst io.Writer, src io.Reader) (err error) {
	b := leakyBuf.Get()
	_, err = io.CopyBuffer(dst, src, b)
	leakyBuf.Put(b)
	return
}

func SetReadTimeout(c net.Conn) {
	c.SetReadDeadline(time.Now().Add(6 * time.Second))
}

func (c *Conn) Write(b []byte) (n int, err error) {

	return
}

func (c *Conn) Read(b []byte) (n int, err error) {

	return
}

func (c *Conn) RemoteIP() (ip string) {
	ip = c.Conn.RemoteAddr().String()
	ip, _, _ = net.SplitHostPort(ip)
	return
}
