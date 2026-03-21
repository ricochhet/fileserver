package logutil

import (
	"fmt"
	"io"
	"sync/atomic"
)

var debug atomic.Bool

// SetDebug sets the debug logging to the specified value.
func SetDebug(v bool) {
	debug.Store(v)
}

// IsDebug gets the debug state.
func IsDebug() bool {
	return debug.Load()
}

// Debug prints if IsDebug is true.
func Debug(w io.Writer, a ...any) {
	if debug.Load() {
		fmt.Fprint(w, "[debug] ")
		fmt.Fprint(w, a...)
	}
}

// Debugf prints if IsDebug is true.
func Debugf(w io.Writer, format string, a ...any) {
	if debug.Load() {
		fmt.Fprintf(w, "[debug] "+format, a...)
	}
}

// Warn prints a warn log.
func Warn(w io.Writer, a ...any) {
	fmt.Fprint(w, "[warn] ")
	fmt.Fprint(w, a...)
}

// Warnf prints a warn log.
func Warnf(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "[warn] "+format, a...)
}

// Error prints an error log.
func Error(w io.Writer, a ...any) {
	fmt.Fprint(w, "[error] ")
	fmt.Fprint(w, a...)
}

// Errorf prints a Error log.
func Errorf(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "[error] "+format, a...)
}

// Info prints an info log.
func Info(w io.Writer, a ...any) {
	fmt.Fprint(w, "[info] ")
	fmt.Fprint(w, a...)
}

// Infof prints a Info log.
func Infof(w io.Writer, format string, a ...any) {
	fmt.Fprintf(w, "[info] "+format, a...)
}
