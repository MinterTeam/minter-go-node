package log

import (
	"github.com/tendermint/tendermint/libs/log"
	"os"
)

var (
	logger log.Logger
)

func init() {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	SetLogger(logger)
}

func SetLogger(l log.Logger) {
	logger = l
}

func Info(msg string, ctx ...interface{}) {
	logger.Info(msg, ctx...)
}

func With(keyvals ...interface{}) log.Logger {
	return logger.With(keyvals...)
}
