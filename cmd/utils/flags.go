package utils

import (
	"os"
	"path/filepath"
)

var (
	MinterHome string
)

func GetMinterHome() string {
	if MinterHome != "" {
		return MinterHome
	}

	home := os.Getenv("MINTERHOME")

	if home != "" {
		return home
	}

	return os.ExpandEnv(filepath.Join("$HOME", ".minter"))
}
