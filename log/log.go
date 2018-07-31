package log

import (
	"github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"
	"os"
)

var (
	logger log.Logger
)

func init() {
	logger, _ := flags.ParseLogLevel("consensus:info,state:info,main:info,*:error", log.NewTMLogger(os.Stdout), "info")
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
