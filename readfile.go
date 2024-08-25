package main

import (
	"io"
	"os"
)

func _readFile(path string) string {
	f, err := os.Open(path)
	_checkf(err, "opening script")
	buf, err := io.ReadAll(f)
	err2 := f.Close()
	if err == nil {
		err = err2
	}
	_checkf(err, "reading script")
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
