package httpasset

import (
	"os"
)

func binself() (*os.File, error) {
	bin, err := os.Open(os.Args[0])
	if err != nil && os.IsNotExist(err) {
		bin, err = os.Open(os.Args[0] + ".exe")
	}
	return bin, err
}
