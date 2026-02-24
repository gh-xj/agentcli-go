package harness

import (
	"errors"
	"fmt"
)

type FailureCode string

const (
	CodeUsage              FailureCode = "usage"
	CodeMissingDependency  FailureCode = "missing_dependency"
	CodeContractValidation FailureCode = "contract_validation"
	CodeExecution          FailureCode = "execution"
	CodeFileIO             FailureCode = "file_io"
	CodeInternal           FailureCode = "internal"
)

const (
	ExitSuccess           = 0
	ExitUsage             = 2
	ExitMissingDependency = 3
	ExitContractFailure   = 4
	ExitExecutionFailure  = 5
	ExitIOFailure         = 6
	ExitInternalFailure   = 7
)

type FailureError struct {
	Failure Failure
	Cause   error
}

func (e *FailureError) Error() string {
	if e == nil {
		return ""
	}
	msg := e.Failure.Message
	if msg == "" {
		msg = "harness failure"
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", msg, e.Cause)
	}
	return msg
}

func (e *FailureError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func NewFailure(code FailureCode, message, hint string, retryable bool) error {
	return &FailureError{
		Failure: Failure{
			Code:      normalizeCode(code),
			Message:   message,
			Hint:      hint,
			Retryable: retryable,
		},
	}
}

func WrapFailure(code FailureCode, message, hint string, retryable bool, cause error) error {
	return &FailureError{
		Failure: Failure{
			Code:      normalizeCode(code),
			Message:   message,
			Hint:      hint,
			Retryable: retryable,
		},
		Cause: cause,
	}
}

func IsCode(err error, code FailureCode) bool {
	var failureErr *FailureError
	if !errors.As(err, &failureErr) {
		return false
	}
	return failureErr.Failure.Code == normalizeCode(code)
}

func ExitCodeFor(err error) int {
	if err == nil {
		return ExitSuccess
	}
	var failureErr *FailureError
	if errors.As(err, &failureErr) {
		return exitCodeForFailureCode(FailureCode(failureErr.Failure.Code))
	}
	return ExitInternalFailure
}

func FailureFromError(err error) Failure {
	if err == nil {
		return Failure{}
	}
	var failureErr *FailureError
	if errors.As(err, &failureErr) {
		f := failureErr.Failure
		f.Code = normalizeCode(FailureCode(f.Code))
		if f.Message == "" {
			f.Message = err.Error()
		}
		return f
	}
	return Failure{
		Code:      normalizeCode(CodeInternal),
		Message:   err.Error(),
		Retryable: false,
	}
}

func normalizeCode(code FailureCode) string {
	switch code {
	case CodeUsage:
		return string(CodeUsage)
	case CodeMissingDependency:
		return string(CodeMissingDependency)
	case CodeContractValidation:
		return string(CodeContractValidation)
	case CodeExecution:
		return string(CodeExecution)
	case CodeFileIO:
		return string(CodeFileIO)
	case CodeInternal:
		return string(CodeInternal)
	default:
		return string(CodeInternal)
	}
}

func exitCodeForFailureCode(code FailureCode) int {
	switch code {
	case CodeUsage:
		return ExitUsage
	case CodeMissingDependency:
		return ExitMissingDependency
	case CodeContractValidation:
		return ExitContractFailure
	case CodeExecution:
		return ExitExecutionFailure
	case CodeFileIO:
		return ExitIOFailure
	case CodeInternal:
		return ExitInternalFailure
	default:
		return ExitInternalFailure
	}
}
