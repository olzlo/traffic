package src

type Auth interface {
	initPasswd() string
}

func NewAuthWithRds() *Auth {

}
func NewAuthWithEnv() *Auth {

}
