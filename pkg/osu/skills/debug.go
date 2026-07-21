//go:build debug

package skills

import "fmt"

func debugf(format string, args ...any) {
	fmt.Printf(format, args...)
}
