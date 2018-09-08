package utils

import (
	"flag"
	"os"
	"os/user"
)

var (
	MinterHome            = flag.String("home", "", "Path to minter data directory")
	ResetPrivateValidator = flag.Bool("reset-private-validator", false, "")
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
