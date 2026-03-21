package logutil

import "github.com/sasha-s/go-deadlock"

type GLog struct {
	mu     deadlock.Mutex
	logger *Logger
}

var global GLog

// Get returns the logger.
func Get() *Logger {
	global.mu.Lock()
	defer global.mu.Unlock()

	return global.logger
}

// Set sets the logger.
func Set(logger *Logger) {
	global.mu.Lock()
	defer global.mu.Unlock()

	global.logger = logger
}

// CopyFrom sets the logger to the target.
func CopyFrom(target *GLog) {
	target.mu.Lock()
	l := target.logger
	target.mu.Unlock()

	global.mu.Lock()
	global.logger = l
	global.mu.Unlock()
}
