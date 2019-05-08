package src

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"os"
	"strings"
)

//Auth user authentication
type IAuth interface {
	init()
	//pre-shared key encrypted handshake
	SharedKey() []byte
	User(string) ([]byte, bool)
}

//NewAuthFromRedis from redis
func NewAuthFromRedis() IAuth {
	return nil
}

//NewAuthFromEnv from env
func NewAuthFromEnv() IAuth {
	return &env{}
}

var _ IAuth = &env{}

type env struct {
}

func (e *env) init() {

}

func randomKeyGen() []byte {
	key := make([]byte, 95)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(err)
	}
	dst := make([]byte, 128)
	base64.StdEncoding.Encode(dst, key)
	return dst
}

func (e *env) SharedKey() (key []byte) {
	if b, ok := os.LookupEnv("TRAFFIC_SHARED"); ok {
		key = []byte(b)
	} else {
		key = randomKeyGen()
		Logger.Info("the pre-shared key has not been set use random key as bellow \n", string(key))
	}
	return
}
func (e *env) GetKey(uname string) (pwd []byte, ok bool) {
	b, ok := os.LookupEnv("TRAFFIC_USER_" + strings.ToUpper(uname))
	pwd = []byte(b)
	return
}
