package logger

import (
	"github.com/rs/zerolog"
	"github.com/telq-org/sitemap-builder/pkg/config"
	"os"
)

var Log zerolog.Logger

func init() {
	level, err := zerolog.ParseLevel(config.Env.LogLevel)
	if err != nil {
		panic(err)
	}

	Log = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(level).With().Timestamp().Caller().Logger()
}

func Must(err error) {
	if err != nil {
		Log.Panic().Err(err).Send()
	}
}

func Err(err error) {
	if err != nil {
		Log.Error().Err(err).Send()
	}
}
