// +build !windows

package httpasset

import (
	"os"
)

func binself() (*os.File, error) {
	return os.Open(os.Args[0])
}
