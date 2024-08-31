package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
)

func cmdBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	var bwrap bool
	var nonet bool
	var clonecmd string
	var toolchainDir string
	fs.BoolVar(&bwrap, "bwrap", false, "isolate environment during build using bubblewrap (bwrap)")
	fs.BoolVar(&nonet, "nonet", false, "execute build without network access; implies bwrap")
	fs.StringVar(&clonecmd, "clone", "", "command to run to clone the repository, instead of looking for .git or .hg in the current directory")
	fs.StringVar(&toolchainDir, "toolchaindir", "", "directory to make available as toolchaindir, e.g. $HOME/sdk for Go toolchains")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ding build [-bwrap] [-nonet] [-clone cmd] [-toolchaindir dir] dstdir buildscript")
		fs.PrintDefaults()
		os.Exit(2)
	}
	fs.Parse(args)
	args = fs.Args()
	if len(args) != 2 {
		fs.Usage()
	}
	if nonet {
		bwrap = true
	}

	workDir, err := os.Getwd()
	xcheckf(err, "get workdir")
	srcdir := workDir

	dstdir := args[0]
	if !path.IsAbs(dstdir) {
		dstdir = path.Join(workDir, dstdir)
	}
	buildscript := args[1]
	if !path.IsAbs(buildscript) {
		buildscript = path.Join(workDir, buildscript)
	}

	// Check that dstdir is absent or empty.
	if files, err := os.ReadDir(dstdir); err == nil && len(files) != 0 {
		xcheckf(errors.New("must be empty"), "checking dstdir")
	} else if err != nil {
		if !os.IsNotExist(err) {
			xcheckf(err, "checking dstdir")
		}
		err := os.MkdirAll(dstdir, 0755)
		xcheckf(err, "making dstdir")
	}

	homeDir := path.Join(dstdir, "home")
	buildDir := path.Join(dstdir, "build")
	var checkoutPath string
	if clonecmd == "" {
		checkoutPath = "checkout"
	} else {
		checkoutPath = path.Base(workDir)
	}

	downloadDir := path.Join(buildDir, "dl")

	// Also see build.go.
	env := []string{
		"PATH=/bin:/usr/bin",
		"HOME=" + homeDir,
		"DING_BUILDDIR=" + buildDir,
		"DING_CHECKOUTPATH=" + checkoutPath,
		"DING_DOWNLOADDIR=" + downloadDir,
		"DING_BUILDID=1",
		"DING_REPONAME=" + checkoutPath,
		// todo: could try to get this from the current checkout
		"DING_BRANCH=ding",
		"DING_COMMIT=deadbeef",
	}
	if toolchainDir != "" {
		env = append(env, "DING_TOOLCHAINDIR="+toolchainDir)
	}

	run := func(build bool, cmdargv ...string) ([]byte, []byte) {
		var argv []string
		if build && bwrap {
			argv = bwrapCmd(nonet, homeDir, buildDir, toolchainDir)
			argv = append(argv, "--bind", buildscript, buildscript)
		}
		argv = append(argv, cmdargv...)
		slog.Info("executing", "cmd", argv, "workdir", buildDir)
		cmd := exec.CommandContext(context.Background(), argv[0], argv[1:]...)
		cmd.Dir = workDir
		cmd.Env = env
		var stdout, stderr bytes.Buffer
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
		err := cmd.Run()
		xcheckf(err, "run command %v", argv)
		return stdout.Bytes(), stderr.Bytes()
	}

	checkoutDir := path.Join(buildDir, "checkout", checkoutPath)

	err = os.MkdirAll(homeDir, 0755)
	xcheckf(err, "making homedir")

	err = os.MkdirAll(path.Dir(checkoutDir), 0755)
	xcheckf(err, "making checkoutdir")

	err = os.MkdirAll(downloadDir, 0755)
	xcheckf(err, "making dl dir")

	// Clone.
	workDir = buildDir
	var historySize int64
	if clonecmd != "" {
		run(false, clonecmd)
	} else if _, err := os.Stat(path.Join(srcdir, ".git")); err == nil {
		run(false, "git", "clone", "--recursive", "--no-hardlinks", srcdir, checkoutDir)
		historySize = buildDiskUsage(path.Join(srcdir, ".git"))
	} else if _, err := os.Stat(path.Join(srcdir, ".hg")); err == nil {
		run(false, "hg", "clone", srcdir, checkoutDir)
		historySize = buildDiskUsage(path.Join(srcdir, ".hg"))
	} else {
		xcheckf(errors.New("cannot find .git or .hg in work dir"), "looking for vcs in workdir")
	}

	checkoutSize := buildDiskUsage(checkoutDir)

	// Build.
	workDir = checkoutDir
	stdout, stderr := run(true, buildscript)
	version, results, coverage, coverageReportFile, err := parseResults(checkoutDir, downloadDir, io.MultiReader(bytes.NewReader(stdout), bytes.NewReader(stderr)))
	xcheckf(err, "parsing results")
	var coverageStr string
	if coverage != nil {
		coverageStr = fmt.Sprintf("%d%%", int(*coverage))
	}
	fmt.Printf("\nbuild ok\nversion %q, coverage %s file %q, %d result(s)\n", version, coverageStr, coverageReportFile, len(results))
	for _, r := range results {
		fmt.Printf("- %#v\n", r)
	}

	homeSize := buildDiskUsage(homeDir)
	buildSize := buildDiskUsage(buildDir)
	const mb = 1024 * 1024
	fmt.Printf("sizes: vcs history %.1fm, checkout %.1fm, home %.1fm, build %.1fm\n", float64(historySize)/mb, float64(checkoutSize-historySize)/mb, float64(homeSize)/mb, float64(buildSize-checkoutSize)/mb)
}
