package agentcli

import (
	"context"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// IOStreams defines where command output should be written.
type IOStreams struct {
	Stdout io.Writer
	Stderr io.Writer
}

// AppMeta carries optional build/runtime metadata.
type AppMeta struct {
	Name    string
	Version string
	Commit  string
	Date    string
}

// AppContext is the shared runtime context for CLI command execution.
type AppContext struct {
	Context context.Context
	Logger  zerolog.Logger
	Config  any
	IO      IOStreams
	Meta    AppMeta
	Values  map[string]any
}

// NewAppContext returns a context with safe defaults for CLI apps.
func NewAppContext(ctx context.Context) *AppContext {
	if ctx == nil {
		ctx = context.Background()
	}
	return &AppContext{
		Context: ctx,
		Logger:  log.Logger,
		IO: IOStreams{
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
		Values: make(map[string]any),
	}
}
