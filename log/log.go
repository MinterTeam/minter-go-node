package log

import (
	"github.com/MinterTeam/minter-go-node/config"
	"github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"
	"io"
	"os"
)

var (
	logger log.Logger
)

func InitLog(cfg *config.Config) {
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

func Fatal(msg string, ctx ...interface{}) {
	logger.Error(msg, ctx...)
	os.Exit(1)
}

func With(keyvals ...interface{}) log.Logger {
	return logger.With(keyvals...)
}
