// Package fault defines errors shared by business operations and the CLI.
package fault

import (
	"errors"
	"fmt"
)

// Error carries a translation key without coupling filesystem code to the UI.
type Error struct {
	Key        string
	Args       []any
	Cause      error
	Code       int
	ClearInput bool
	Stop       bool
	Warning    bool
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Key, e.Cause)
	}
	return fmt.Sprint(append([]any{e.Key, ": "}, e.Args...)...)
}

func (e *Error) Unwrap() error { return e.Cause }

func New(key string, args ...any) *Error {
	stop := key == "changed" || key == "initPartial" || key == "renamePartial" || key == "removePartial"
	warning := key == "elevationCancelled" || key == "elevationTerminal" || key == "sourceKept" || key == "busy" || key == "cancelled"
	return &Error{Key: key, Args: args, Code: 1, Stop: stop, Warning: warning}
}

func Wrap(key string, err error, args ...any) *Error {
	e := New(key, args...)
	e.Cause = err
	return e
}

func ExitCode(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return 1
}

func IsKey(err error, key string) bool {
	var e *Error
	return errors.As(err, &e) && e.Key == key
}

func IsWarning(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Warning
}
