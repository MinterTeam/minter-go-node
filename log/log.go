package log

import "github.com/tendermint/tendermint/libs/log"

var (
	logger log.Logger
)

func SetLogger(l log.Logger) {
	logger = l
}

func Info(msg string, ctx ...interface{}) {
	logger.Info(msg, ctx...)
}
