package logging

import (
	"os"

	"github.com/rs/zerolog"
)

func New() *zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &log
}
