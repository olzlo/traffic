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

//NewAuthWithRds from redis
func NewAuthWithRds() *IAuth {
	return nil
}

//NewAuthWithEnv from env
func NewAuthWithEnv() *IAuth {
	return nil
}

var _ IAuth = &env{}

type env struct {
}

func (e *env) init() {

}

func (e *env) SharedKey() (key []byte, ok bool) {
	b, ok := os.LookupEnv("TRAFFIC_SHARED")
	key=string(b)
	return
}
func (e *env) User(uname string) (pwd []byte, ok bool) {
	b, ok := os.LookupEnv("TRAFFIC_USER_" + strings.ToUpper(uname))
	pwd = string(b)
	return
}
