package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func New(component string) zerolog.Logger {
	sublogger := log.With().
		Str("component", component).
		Logger()
	return sublogger
}

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	_, debug := os.LookupEnv("DEBUG")
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	})
}
