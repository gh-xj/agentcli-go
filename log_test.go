package agentcli

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestInitLoggerDefaultLevel(t *testing.T) {
	oldArgs := os.Args
	oldLogger := log.Logger
	defer func() {
		os.Args = oldArgs
		log.Logger = oldLogger
	}()

	os.Args = []string{"cmd"}
	InitLogger()
	if got := log.Logger.GetLevel(); got != zerolog.InfoLevel {
		t.Fatalf("expected info level, got %s", got.String())
	}
}

func TestInitLoggerVerboseLevel(t *testing.T) {
	oldArgs := os.Args
	oldLogger := log.Logger
	defer func() {
		os.Args = oldArgs
		log.Logger = oldLogger
	}()

	os.Args = []string{"cmd", "--verbose"}
	InitLogger()
	if got := log.Logger.GetLevel(); got != zerolog.DebugLevel {
		t.Fatalf("expected debug level, got %s", got.String())
	}
}
