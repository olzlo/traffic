package src

import (
	"crypto/cipher"
)

type Cipher struct {
	stream cipher.Stream
	key    []byte
}

func (c *Cipher) SetKey(key []byte) {
	c.key = key
}
