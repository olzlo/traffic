package src

import (
	"os"
	"strings"
)

//Auth user authentication
type IAuth interface {
	init()
	//pre-shared key encrypted handshake
	SharedKey() ([]byte, bool)
	User(string) ([]byte, bool)
}

//NewAuthFromRds from redis
func NewAuthFromRds() IAuth {
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

func (e *env) SharedKey() (key []byte, ok bool) {
	b, ok := os.LookupEnv("TRAFFIC_SHARED")
	key = []byte(b)
	return
}
func (e *env) User(uname string) (pwd []byte, ok bool) {
	b, ok := os.LookupEnv("TRAFFIC_USER_" + strings.ToUpper(uname))
	pwd = []byte(b)
	return
}
