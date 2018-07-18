package utils

import (
	"flag"
	"os"
	"os/user"
)

var (
	TendermintRpcAddrFlag = flag.String("tendermint_addr", "tcp://0.0.0.0:46657", "This is the address that minter will use to connect to the tendermint core node. Please provide a port.")
	MinterAPIAddrFlag     = flag.String("api_addr", ":8841", "This is the address that minter will use to open API server. Please provide a port.")
	MinterHome            = flag.String("home", "", "Path to minter data directory")
	DisableApi            = flag.Bool("disable-api", false, "")
)

func init() {
	flag.Parse()
}

func GetMinterHome() string {

	if *MinterHome != "" {
		return *MinterHome
	}

	home := os.Getenv("MINTERHOME")

	if home != "" {
		return home
	}

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	return usr.HomeDir + "/.minter"
}
