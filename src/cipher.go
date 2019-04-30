package src

import (
	"crypto/cipher"
)

type Cipher struct {
	stream cipher.Stream
}
