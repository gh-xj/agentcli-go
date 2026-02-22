package agentcli

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// InitLogger sets up zerolog with a console writer on stderr.
// Reads -v/--verbose from os.Args to enable debug-level output.
func InitLogger() {
	verbose := lo.Contains(os.Args, "-v") || lo.Contains(os.Args, "--verbose")
	level := lo.Ternary(verbose, zerolog.DebugLevel, zerolog.InfoLevel)
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		NoColor:    false,
		TimeFormat: "15:04:05",
	}).With().Timestamp().Logger().Level(level)
}
