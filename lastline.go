package main

import (
	"bufio"
	"fmt"
	"os"
)

func setLastLine(b *Build) {
	// todo: should just try reading from build.output, then clone.output.

	var status BuildStatus
	switch b.Status {
	case StatusClone:
		status = b.Status
	case StatusBuild, StatusSuccess:
		status = StatusBuild
	default:
		b.LastLine = ""
		return
	}
	b.LastLine = ""

	path := fmt.Sprintf("%s/build/%s/%d/output/%s.output", dingDataDir, b.RepoName, b.ID, status)
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			b.LastLine = fmt.Sprintf("(open for last line: %s)", err)
		}
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		if s != "" {
			b.LastLine = s
		}
	}
	if err = scanner.Err(); err != nil {
		b.LastLine = fmt.Sprintf("(reading for last line: %s)", err)
	}
}
