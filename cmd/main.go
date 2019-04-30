package main

import (
	"flag"
	mo "github.com/zsichen/traffic/src"
)

type Command struct {
	LocalPort string
	Rds       string
}

var (
	comm Command
	auth mo.Auth
)

func main() {
	flag.StringVar(&comm.LocalPort, "l", ":10020", "listen port")
	flag.StringVar(&comm.Rds, "rds", "", "use redis store user's info")
	flag.Parse()

	if comm.Rds == "" {
		auth = mo.NewAuthWithEnv()
	} else {
		auth = mo.NewAuthWithRds()
	}

}
