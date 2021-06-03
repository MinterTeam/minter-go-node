package log

import (
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"
	"io"
	"os"
)

// NewLogger returns a logger based on given config
func NewLogger(cfg *config.Config) log.Logger {
	var dest io.Writer = os.Stdout

	if cfg.LogPath != "stdout" {
		file, err := os.OpenFile(cfg.LogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

		if err != nil {
			panic(err)
		}

		dest = file
	}

	var l log.Logger

	switch cfg.LogFormat {
	case config.LogFormatJSON:
		l = log.NewTMJSONLogger(dest)
	case config.LogFormatPlain:
		l = log.NewTMLogger(dest)
	default:
		panic("unsupported log format")
	}

	l, err := flags.ParseLogLevel(cfg.LogLevel, l, "info")

	if err != nil {
		panic(err)
	}

	return l
}
