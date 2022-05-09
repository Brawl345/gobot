package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	*zerolog.Logger
}

func New(component string) *Logger {
	sublogger := log.With().
		Str("component", component).
		Logger()
	return &Logger{&sublogger}
}

func init() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	_, debug := os.LookupEnv("DEBUG")
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	_, prettyPrint := os.LookupEnv("PRETTY_PRINT_LOG")
	if prettyPrint {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		})
	}
}
