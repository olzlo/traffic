package src

import (
	"crypto/cipher"
	"io"
	"net"
	"time"
)

const (
	TransferBufferSize = 4096
)

type Conn struct {
	net.Conn
	*Cipher
	Token string
}

func NewEncryptConn(c net.Conn, key []byte, stream func(key, iv []byte) (cipher.Stream, error)) *Conn {
	return &Conn{
		c, &Cipher{
			key:       key,
			newStream: stream,
		}, "",
	}
}

func Copy(dst io.Writer, src io.Reader) (err error) {
	b := BufferPool.Get(TransferBufferSize)
	_, err = CopyBuffer(dst, src, b)
	BufferPool.Put(b)
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
		defer BufferPool.Put(iv)
		if n, err = c.Conn.Write(iv); err != nil {
			return
		}
	}
	cipherData := BufferPool.Get(TransferBufferSize)
	c.encrypt(cipherData, b)
	n, err = c.Conn.Write(cipherData[:len(b)])
	BufferPool.Put(cipherData)
	return
}

func (c *Conn) Read(b []byte) (n int, err error) {
	if c.dec == nil {
		iv := BufferPool.Get(8)
		defer BufferPool.Put(iv)
		if _, err = io.ReadFull(c.Conn, iv); err != nil {
			return
		}
		if err = c.initDecrypt(iv); err != nil {
			return
		}
	}
	cipherData := BufferPool.Get(TransferBufferSize)
	n, err = c.Conn.Read(cipherData[:len(b)])
	if n > 0 {
		c.decrypt(b[:n], cipherData[:n])
	}
	BufferPool.Put(cipherData)
	return
}

func RemoteIP(conn net.Conn) (ip string) {
	ip = conn.RemoteAddr().String()
	ip, _, _ = net.SplitHostPort(ip)
	return
}
