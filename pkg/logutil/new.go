package logutil

// NewLogger is a convenience function for CreateLogger that additionally assigns MaxProcNameLength.
func NewLogger(name string, colorIndex int) *Logger {
	MaxProcNameLength.Store(int32(len(name)))
	return CreateLogger(name, colorIndex)
}
