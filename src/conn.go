package src

import (
	"crypto/cipher"
	"io"
	"net"
	"time"
)

type Conn struct {
	net.Conn
	*Cipher
}

func NewEncryptConn(c net.Conn, key []byte, stream func(key, iv []byte) (cipher.Stream, error)) *Conn {
	return &Conn{
		c, &Cipher{
			key:       key,
			newStream: stream,
		},
	}
}

func Copy(dst io.Writer, src io.Reader) (err error) {
	b := bufPool.Get()
	_, err = io.CopyBuffer(dst, src, b)
	bufPool.Put(b)
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

func RemoteIP(conn net.Conn) (ip string) {
	ip = conn.RemoteAddr().String()
	ip, _, _ = net.SplitHostPort(ip)
	return
}
