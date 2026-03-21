//go:build !windows
// +build !windows

package cmdutil

// QuickEdit sets quick edit according to the specified value.
func QuickEdit(_ bool) error {
	return nil
}
