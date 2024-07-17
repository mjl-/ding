package main

import (
	"io"
	"os"
)

func readFile(path string) string {
	f, err := os.Open(path)
	sherpaCheck(err, "opening script")
	buf, err := io.ReadAll(f)
	err2 := f.Close()
	if err == nil {
		err = err2
	}
	sherpaCheck(err, "reading script")
	return string(buf)
}

func readFileLax(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	buf, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return ""
	}
	return string(buf)
}
