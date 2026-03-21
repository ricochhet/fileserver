//go:build !windows
// +build !windows

package winutil

// https://gist.github.com/jerblack/d0eb182cc5a1c1d92d92a4c4fcc416c6
// isAdmin checks if the current process is ran as administrator.
func IsAdmin() bool {
	return true
}
