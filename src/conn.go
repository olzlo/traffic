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
	var iv []byte
	if c.enc == nil {
		iv, err = c.initEncrypt()
		if err != nil {
			return
		}
		if n, err = c.Conn.Write(iv); err != nil {
			return
		}
	}
	cipherData := bufPool.Get()
	c.encrypt(cipherData, b)
	n, err = c.Conn.Write(cipherData)
	bufPool.Put(cipherData)
	return
}

func (c *Conn) Read(b []byte) (n int, err error) {
	if c.dec == nil {
		iv := make([]byte, 8)
		if _, err = io.ReadFull(c.Conn, iv); err != nil {
			return
		}
		if err = c.initDecrypt(iv); err != nil {
			return
		}
	}
	cipherData := bufPool.Get()
	n, err = c.Conn.Read(cipherData)
	if n > 0 {
		c.decrypt(b[:n], cipherData[:n])
	}
	bufPool.Put(cipherData)
	return
}

func RemoteIP(conn net.Conn) (ip string) {
	ip = conn.RemoteAddr().String()
	ip, _, _ = net.SplitHostPort(ip)
	return
}
