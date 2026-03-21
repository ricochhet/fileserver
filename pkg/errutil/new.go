package errutil

import (
	"fmt"
	"runtime"
)

// Newf returns a formatted error with the frame of the caller.
func Newf(format, errFormat string, errA ...any) error {
	return newErr(format, fmt.Errorf(errFormat, errA...), 2)
}

// New returns a formatted error with the frame of the caller.
func New(format string, err error, a ...any) error {
	return newErr(format, err, 2, a...)
}

// newErr returns a formatted error with the frame of the caller.
func newErr(format string, err error, skip int, a ...any) error {
	if err == nil {
		return nil
	}

	_, f, l, ok := runtime.Caller(skip)
	if !ok {
		return err
	}

	args := []any{f, l}
	if len(a) > 0 {
		args = append(args, a...)
		args = append(args, err)
	} else {
		args = append(args, err)
	}

	return fmt.Errorf("%s (%d): "+format+": %w", args...)
}
