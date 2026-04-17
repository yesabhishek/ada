package cli

import (
	"errors"
	"fmt"
)

const (
	exitRuntime     = 1
	exitUsage       = 2
	exitEnvironment = 3
	exitDependency  = 4
)

type commandError struct {
	code    int
	message string
	cause   error
}

func (e *commandError) Error() string {
	if e.cause == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %v", e.message, e.cause)
}

func (e *commandError) Unwrap() error {
	return e.cause
}

func usageErrorf(format string, args ...any) error {
	return &commandError{code: exitUsage, message: fmt.Sprintf(format, args...)}
}

func environmentErrorf(format string, args ...any) error {
	return &commandError{code: exitEnvironment, message: fmt.Sprintf(format, args...)}
}

func dependencyErrorf(format string, args ...any) error {
	return &commandError{code: exitDependency, message: fmt.Sprintf(format, args...)}
}

func runtimeError(message string, err error) error {
	return &commandError{code: exitRuntime, message: message, cause: err}
}

func ExitCode(err error) int {
	var commandErr *commandError
	if errors.As(err, &commandErr) {
		return commandErr.code
	}
	return exitRuntime
}
