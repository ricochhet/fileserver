package sliceutil

// Or returns v if it's not empty, otherwise return or.
func Or[T any](v, or []T) []T {
	if len(v) == 0 {
		return or
	}

	return v
}
