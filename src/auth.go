package src

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"io"
	"os"
	"strings"

	rds "github.com/go-redis/redis"
)

//Auth user authentication
type IAuth interface {
	//pre-shared key encrypted handshake
	SharedKey() []byte
	IsValid(string) bool
}

//NewAuthFromRedis from redis
func NewAuthFromRedis(addr string) IAuth {
	return &redisAuth{
		cli: rds.NewClient(&rds.Options{
			Addr: addr,
		}),
	}
}

//NewAuthFromEnv from env
func NewAuthFromEnv() IAuth {
	e := &envAuth{}
	if b, ok := os.LookupEnv("TRAFFIC_SHARED"); ok {
		e.key = EnforceKeys([]byte(b), 32)
	} else {
		key := randomKeyGen(16)
		e.key = EnforceKeys(key, 32)
		Logger.Info("random pre-shared key : ", string(key))
	}
	return e
}

var _ IAuth = &envAuth{}
var _ IAuth = &redisAuth{}

func randomKeyGen(keyLen int) []byte {
	key := make([]byte, keyLen)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		panic(err)
	}
	dst := make([]byte, keyLen*2)
	base64.StdEncoding.Encode(dst, key)
	return dst[:keyLen]
}

func EnforceKeys(unsafe []byte, keyLen int) (key []byte) {
	key = make([]byte, keyLen)
	b := make([]byte, md5.BlockSize+len(unsafe))
	hash := md5.Sum(unsafe)

	copy(b, []byte(hash[:]))
	copy(b[md5.BlockSize:], unsafe)

	for i := 0; i < keyLen; i++ {
		key[i] = b[0]
		hash = md5.Sum(b)
		copy(b, []byte(hash[:]))
		copy(b[md5.BlockSize:], unsafe)
	}
	return
}

type redisAuth struct {
	cli *rds.Client
}

func (r *redisAuth) SharedKey() (key []byte) {
	val, err := r.cli.Get("traffic:shared").Result()
	if err != nil {
		panic(err)
	}
	return EnforceKeys([]byte(val), 32)
}

func (r *redisAuth) IsValid(token string) (ok bool) {
	ok, err := r.cli.SIsMember("traffic:token", token).Result()
	if err != nil {
		panic(err)
	}
	return
}

type envAuth struct {
	key []byte
}

func (e *envAuth) SharedKey() []byte {
	if b, ok := os.LookupEnv("TRAFFIC_SHARED"); ok {
		e.key = EnforceKeys([]byte(b), 32)
	}
	return e.key
}
func (e *envAuth) IsValid(token string) (ok bool) {
	_, ok = os.LookupEnv("TRAFFIC_USER_" + strings.ToUpper(token))
	return
}
