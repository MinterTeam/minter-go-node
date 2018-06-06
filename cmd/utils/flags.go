package utils

import (
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/user"
)

var (
	TendermintRpcAddrFlag = cli.StringFlag{
		Name:  "tendermint_addr",
		Value: "tcp://127.0.0.1:46657",
		Usage: "This is the address that minter will use to connect to the tendermint core node. Please provide a port.",
	}
)

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
