package timeutil

import (
	"time"

	"github.com/ricochhet/fileserver/pkg/errutil"
)

// TimerWithResult starts a timer with a function that returns a result (T).
func TimerWithResult[T any](
	fn func() (T, error),
	name string,
	caller func(string, string),
) (T, error) {
	start := time.Now()
	result, err := fn()
	elapsed := time.Since(start)
	caller(name, elapsed.String())

	return result, errutil.WithFrame(err)
}

// Timer starts a timer with a function.
func Timer(fn func() error, name string, caller func(string, string)) error {
	start := time.Now()
	err := fn()
	elapsed := time.Since(start)
	caller(name, elapsed.String())

	return errutil.WithFrame(err)
}
