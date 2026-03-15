package dal

import (
	"io"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LoggerImpl is the real zerolog-backed Logger.
type LoggerImpl struct{}

// NewLogger returns a new LoggerImpl.
func NewLogger() *LoggerImpl { return &LoggerImpl{} }

func (l *LoggerImpl) Init(verbose bool, w io.Writer) {
	level := zerolog.InfoLevel
	if verbose {
		level = zerolog.DebugLevel
	}
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:        w,
		NoColor:    false,
		TimeFormat: "15:04:05",
	}).With().Timestamp().Logger().Level(level)
}
