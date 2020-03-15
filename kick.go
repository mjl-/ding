package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mjl-/sherpa/client"
)

func kick(args []string) {
	fs := flag.NewFlagSet("kick", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ding kick baseURL password repoName branch commit")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	args = fs.Args()
	if len(args) != 4 {
		fs.Usage()
		os.Exit(2)
	}

	baseURL := args[0]
	password := args[1]
	repoName := args[2]
	branch := args[3]
	commit := args[4]

	client, err := client.New(baseURL, []string{"build"})
	check(err, "initializing sherpa client")

	var build struct {
		ID int64
	}
	err = client.Call(context.Background(), &build, "createBuild", password, repoName, branch, commit)
	check(err, "building")
	_, err = fmt.Println("buildId", build.ID)
	check(err, "write")
}
