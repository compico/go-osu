//go:build darwin || freebsd || linux || netbsd || openbsd

package service

// TODO If anyone uses wine, please suggest a solution to this problem. I'd be very grateful for a PR
func getPathFromRegistry() (string, error) {
	return "", nil
}
