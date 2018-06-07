package utils

import (
	"flag"
	"os"
	"os/user"
)

var (
	TendermintRpcAddrFlag = flag.String("tendermint_addr", "tcp://0.0.0.0:46657", "This is the address that minter will use to connect to the tendermint core node. Please provide a port.")
)

func init() {
	flag.Parse()
}

func GetMinterHome() string {
	home := os.Getenv("MINTERHOME")

	if home == "" {
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}
		home = usr.HomeDir + "/.minter"
	}

	return home
}
