package src

import (
	"crypto/cipher"
)

type Cipher struct {
	stream    cipher.Stream
	key       []byte
	iv        []byte
	newStream func(key, iv []byte) (cipher.Stream, error)
}
