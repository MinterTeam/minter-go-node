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
	logger, _ := flags.ParseLogLevel(cfg.LogLevel, log.NewTMLogger(os.Stdout), "info")
	SetLogger(logger)
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
