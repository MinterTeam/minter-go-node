package log

import (
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"
	"os"
)

var (
	logger log.Logger
	cfg    = config.GetConfig()
)

func init() {
	var l log.Logger

	if cfg.LogPath != "stdout" {
		file, err := os.OpenFile(cfg.LogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

		if err != nil {
			panic(err)
		}

		l = log.NewTMLogger(file)
	} else {
		l = log.NewTMLogger(os.Stdout)
	}

	l, err := flags.ParseLogLevel(cfg.LogLevel, l, "info")

	if err != nil {
		panic(err)
	}

	SetLogger(l)
}

func SetLogger(l log.Logger) {
	logger = l
}

func Info(msg string, ctx ...interface{}) {
	logger.Info(msg, ctx...)
}

func Error(msg string, ctx ...interface{}) {
	logger.Error(msg, ctx...)
}

func With(keyvals ...interface{}) log.Logger {
	return logger.With(keyvals...)
}
