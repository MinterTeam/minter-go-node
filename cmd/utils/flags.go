package utils

import (
	"flag"
	"os"
	"path/filepath"
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

	return os.ExpandEnv(filepath.Join("$HOME", ".minter"))
}
