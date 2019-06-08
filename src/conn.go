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

func Pipe(client net.Conn, server net.Conn) (wc, rs int, err error) {
	ch := make(chan error)
	go func() {
		n, err := io.Copy(client, server)
		client.SetDeadline(time.Now())
		server.SetDeadline(time.Now())
		wc = int(n)
		ch <- err
	}()
	n, err := io.Copy(server, client)
	client.SetDeadline(time.Now())
	server.SetDeadline(time.Now())
	rs = int(n)
	ce := <-ch
	if err == nil {
		err = ce
	}
	if err, ok := err.(net.Error); ok && err.Timeout() {
		err = nil
	}
	return
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
	cipherData := make([]byte, len(b))
	c.encrypt(cipherData, b)
	n, err = c.Conn.Write(cipherData[:len(b)])
	return
}

func (c *Conn) Read(b []byte) (n int, err error) {
	if c.dec == nil {
		var iv [8]byte
		if _, err = io.ReadFull(c.Conn, iv[:]); err != nil {
			return
		}
		if err = c.initDecrypt(iv[:]); err != nil {
			return
		}
	}
	cipherData := make([]byte, len(b))
	n, err = c.Conn.Read(cipherData[:len(b)])
	if n > 0 {
		c.decrypt(b[:n], cipherData[:n])
	}
	return
}
