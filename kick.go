package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mjl-/sherpa/client"
)

func kick(args []string) {
	fs := flag.NewFlagSet("kick", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ding kick baseURL repoName branch commit < password-file")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	args = fs.Args()
	if len(args) != 4 {
		fs.Usage()
		os.Exit(2)
	}

	baseURL := args[0]
	repoName := args[1]
	branch := args[2]
	commit := args[3]
	buf, err := io.ReadAll(os.Stdin)
	xcheckf(err, "reading password from stdin")
	password := strings.TrimRight(string(buf), "\n")

	client, err := client.New(baseURL, []string{"build"})
	xcheckf(err, "initializing sherpa client")

	var build struct {
		ID int64
	}
	err = client.Call(context.Background(), &build, "CreateBuild", password, repoName, branch, commit, false)
	xcheckf(err, "building")
	_, err = fmt.Println("buildId", build.ID)
	xcheckf(err, "write")
}
