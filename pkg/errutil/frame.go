package errutil

import (
	"fmt"
	"runtime"
)

// WithFrame returns an error with the frame of the caller.
func WithFramef(format string, a ...any) error {
	return newErrWithFrame(fmt.Errorf(format, a...), 2)
}

// WithFrame returns an error with the frame of the caller.
func WithFrame(err error) error {
	return newErrWithFrame(err, 2)
}

// newErrWithFrame returns an error with the frame of the caller.
func newErrWithFrame(err error, skip int) error {
	if err == nil {
		return nil
	}

	_, f, l, ok := runtime.Caller(skip)
	if !ok {
		return err
	}

	return fmt.Errorf("%s (%d): %w", f, l, err)
}
