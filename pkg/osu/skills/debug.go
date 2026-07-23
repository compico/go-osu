//go:build debug

package skills

import "fmt"

const debug = true

func debugf(format string, args ...any) {
	fmt.Printf(format, args...)
}
