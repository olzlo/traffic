package src

import (
	"crypto/cipher"
	"crypto/rand"
	"io"
)

type Cipher struct {
	dec       cipher.Stream
	enc       cipher.Stream
	key       []byte
	newStream func(iv, key []byte) (cipher.Stream, error)
}

func (c *Cipher) initEncrypt() (iv []byte, err error) {
	iv = BufferPool.Get(8)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	c.enc, err = c.newStream(iv, c.key)
	return
}

func (c *Cipher) initDecrypt(iv []byte) (err error) {
	c.dec, err = c.newStream(iv, c.key)
	return
}

func (c *Cipher) encrypt(dst, src []byte) {
	c.enc.XORKeyStream(dst, src)
}

func (c *Cipher) decrypt(dst, src []byte) {
	c.dec.XORKeyStream(dst, src)
}
