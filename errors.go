package agentcli

import (
	"errors"
	"fmt"
)

const (
	ExitSuccess             = 0
	ExitFailure             = 1
	ExitUsage               = 2
	ExitPreflightDependency = 3
	ExitRuntimeExternal     = 4
)

// ExitCoder describes errors that can provide a process exit code.
type ExitCoder interface {
	error
	ExitCode() int
}

// CLIError is a typed error with a deterministic exit code.
type CLIError struct {
	Code    int
	Kind    string
	Message string
	Cause   error
}

func (e *CLIError) Error() string {
	if e == nil {
		return ""
	}
	base := e.Message
	if e.Kind != "" {
		base = fmt.Sprintf("%s: %s", e.Kind, e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", base, e.Cause)
	}
	return base
}

func (e *CLIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *CLIError) ExitCode() int {
	if e == nil || e.Code <= 0 {
		return ExitFailure
	}
	return e.Code
}

// NewCLIError builds a typed CLI error with an optional cause.
func NewCLIError(code int, kind, message string, cause error) *CLIError {
	return &CLIError{
		Code:    code,
		Kind:    kind,
		Message: message,
		Cause:   cause,
	}
}

// ResolveExitCode maps an error to a deterministic exit code.
func ResolveExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var coded ExitCoder
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return ExitFailure
}
