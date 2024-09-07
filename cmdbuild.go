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
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"time"
)

func cmdBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	var nobwrap, needbwrap bool
	var goauto, xgo, goprev, gonext bool
	var nonet bool
	var clonecmd string
	var toolchainDir string
	sdkDir := filepath.Join(os.Getenv("HOME"), "sdk")
	if _, err := os.Stat(sdkDir); err == nil {
		toolchainDir = sdkDir
	}
	var destdir string
	var tmpdestdir bool

	fs.BoolVar(&nobwrap, "nobwrap", false, "don't use bwrap; automatically used if available otherwise")
	fs.BoolVar(&needbwrap, "needbwrap", false, "require bwrap, failing if not available")
	fs.BoolVar(&nonet, "nonet", false, "execute build without network access; implies -needbwrap")
	fs.BoolVar(&goauto, "goauto", false, "execute build script for available go toolchains: go, goprev, gonext, as available in toolchaindir (you may want to make symlinks there)")
	fs.BoolVar(&xgo, "go", false, "execute build script \"go\" toolchain")
	fs.BoolVar(&goprev, "goprev", false, "execute build script \"goprev\" toolchain")
	fs.BoolVar(&gonext, "gonext", false, "execute build script \"gonext\" toolchain")
	fs.StringVar(&destdir, "destdir", "", "directory for build, must be empty or not exist; if not specified, a tmpdir is automatically created and removed after the build")
	fs.StringVar(&clonecmd, "clone", "", "command to run to clone the repository, instead of looking for .git or .hg in the current directory")
	fs.StringVar(&toolchainDir, "toolchaindir", toolchainDir, "directory to make available as toolchaindir; if $HOME/sdk exists, it is used as toolchaindir")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ding build [-goauto] [-go] [-goprev] [-gonext] [-nobwrap] [-needbwrap] [-nonet] [-clone cmd] [-toolchaindir dir] [-destdir dir] cmd ...")
		fs.PrintDefaults()
		os.Exit(2)
	}
	fs.Parse(args)
	args = fs.Args()
	if len(args) < 1 {
		fs.Usage()
	}
	if nobwrap && needbwrap {
		slog.Error("-nobwrap and -needbwrap are incompatible")
		flag.Usage()
	}
	if nonet {
		needbwrap = true
	}
	if goauto && (xgo || goprev || gonext) {
		slog.Error("-goauto and the other -go* flags should not be used together")
		flag.Usage()
	}
	if toolchainDir == "" && (goauto || xgo || goprev || gonext) {
		slog.Error("-toolchaindir required for -go* flags")
		os.Exit(1)
	}
	if goauto {
		statGo := func(name string) bool {
			_, err := os.Stat(filepath.Join(toolchainDir, name))
			return err == nil
		}
		xgo = statGo("go")
		goprev = statGo("goprev")
		gonext = statGo("gonext")
		if !xgo && !goprev && !gonext {
			slog.Error("no go toolchains found, you may want to make symlinks named go, goprev and/or gonext in the toolchaindir")
			os.Exit(1)
		}
	}

	workDir, err := os.Getwd()
	xcheckf(err, "get workdir")
	srcdir := workDir

	cleanupTempDestDir := func() {
		if tmpdestdir {
			err := os.RemoveAll(destdir)
			if err != nil {
				slog.Error("removing temporary destdir", "err", err, "destdir", destdir)
			}
		}
	}

	ctx, ctxcancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		ctxcancel()
		slog.Error("signal, cleaning up and stopping")
		// Give build process time to stop.
		time.Sleep(50 * time.Millisecond)
		cleanupTempDestDir()
		os.Exit(1)
	}()

	// From here on, we don't use xcheckf, only xlcheckf: it cleans up any temp destdir.
	xlcheckf := func(err error, format string, args ...any) {
		if err == nil {
			return
		}

		msg := fmt.Sprintf(format, args...)
		slog.Error(msg, "err", err)

		cleanupTempDestDir()

		os.Exit(1)
	}

	if destdir != "" {
		if !path.IsAbs(destdir) {
			destdir = path.Join(workDir, destdir)
		}

		// Check that destdir is absent or empty.
		if files, err := os.ReadDir(destdir); err == nil && len(files) != 0 {
			xlcheckf(errors.New("must be empty"), "checking destdir")
		} else if err != nil {
			if !os.IsNotExist(err) {
				xlcheckf(err, "checking destdir")
			}
			err := os.MkdirAll(destdir, 0755)
			xlcheckf(err, "making destdir")
		}
	} else {
		destdir, err = os.MkdirTemp("", "ding-build")
		xlcheckf(err, "making temp destdir")
		tmpdestdir = true
		slog.Info("tempdir created, will be removed", "destdir", destdir)
	}
	buildscript := args[0]
	var bindBuildscript bool
	if _, err := os.Stat(buildscript); err == nil {
		bindBuildscript = true
		if !path.IsAbs(buildscript) {
			buildscript = path.Join(workDir, buildscript)
		}
	}

	homeDir := path.Join(destdir, "home")
	buildDir := path.Join(destdir, "build")
	var checkoutPath string
	if clonecmd == "" {
		checkoutPath = "checkout"
	} else {
		checkoutPath = path.Base(workDir)
	}

	downloadDir := path.Join(buildDir, "dl")

	// Also see build.go.
	environment := []string{
		"PATH=/usr/bin:/bin:/usr/local/bin", // Must be first, may be changed.
		"HOME=/home/ding",
		"DING_BUILDDIR=/home/ding/build",
		"DING_CHECKOUTPATH=" + checkoutPath,
		"DING_DOWNLOADDIR=/home/ding/build/dl",
		"DING_BUILDID=1",
		"DING_REPONAME=" + checkoutPath,
		// todo: could try to get this from the current checkout
		"DING_BRANCH=ding",
		"DING_COMMIT=deadbeef",
	}
	if toolchainDir != "" {
		environment = append(environment, "DING_TOOLCHAINDIR=/home/ding/toolchain")
	}

	run := func(build bool, env []string, cmdargv ...string) ([]byte, []byte) {
		var argv []string
		if build && (needbwrap || (!nobwrap && hasBubblewrap(ctx))) {
			argv = bwrapCmd(nonet, homeDir, buildDir, checkoutPath, toolchainDir)
			if bindBuildscript {
				dstbuildscript := buildscript
				if strings.HasPrefix(buildscript, workDir+"/") {
					dstbuildscript = strings.Replace(dstbuildscript, workDir, "/home/ding/build/checkout/"+checkoutPath, 1)
					buildscript = strings.Replace(buildscript, workDir, "/home/ding/build/checkout/"+checkoutPath, 1)
				}
				argv = append(argv, "--bind", buildscript, dstbuildscript)
			}
		}
		argv = append(argv, cmdargv...)
		slog.Info("executing", "cmd", argv, "workdir", buildDir)
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		cmd.Dir = workDir
		cmd.Env = env
		var stdout, stderr bytes.Buffer
		cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
		err := cmd.Run()
		xlcheckf(err, "run command %v", argv)
		return stdout.Bytes(), stderr.Bytes()
	}

	checkoutDir := path.Join(buildDir, "checkout", checkoutPath)

	err = os.MkdirAll(homeDir, 0755)
	xlcheckf(err, "making homedir")

	err = os.MkdirAll(path.Dir(checkoutDir), 0755)
	xlcheckf(err, "making checkoutdir")

	err = os.MkdirAll(downloadDir, 0755)
	xlcheckf(err, "making dl dir")

	// Clone.
	workDir = buildDir
	var historySize int64
	if clonecmd != "" {
		run(false, environment, "sh", "-c", clonecmd)
	} else if _, err := os.Stat(path.Join(srcdir, ".git")); err == nil {
		run(false, environment, "git", "clone", "--recursive", "--no-hardlinks", srcdir, checkoutDir)
		historySize = buildDiskUsage(path.Join(srcdir, ".git"))
	} else if _, err := os.Stat(path.Join(srcdir, ".hg")); err == nil {
		run(false, environment, "hg", "clone", srcdir, checkoutDir)
		historySize = buildDiskUsage(path.Join(srcdir, ".hg"))
	} else {
		xlcheckf(errors.New("cannot find .git or .hg in work dir"), "looking for vcs in workdir")
	}

	checkoutSize := buildDiskUsage(checkoutDir)

	// Build.
	build := func(env []string) {
		workDir = checkoutDir
		args[0] = buildscript
		stdout, stderr := run(true, env, args...)
		version, results, coverage, coverageReportFile, err := parseResults(checkoutDir, downloadDir, io.MultiReader(bytes.NewReader(stdout), bytes.NewReader(stderr)))
		xlcheckf(err, "parsing results")
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

	if xgo || goprev || gonext {
		buildGo := func(goname string) {
			cmd := exec.CommandContext(ctx, filepath.Join(toolchainDir, goname, "bin", "go"), "version")
			var stdout strings.Builder
			cmd.Stdout = &stdout
			err := cmd.Run()
			xlcheckf(err, "running go version")
			goverstr := stdout.String()
			t := strings.Split(goverstr, " ")
			if len(t) < 3 || !strings.HasPrefix(t[2], "go1.") {
				xlcheckf(fmt.Errorf("unexpected output %q", goverstr), "parsing output of go version")
			}
			goversion := t[2]

			env := append([]string{}, environment...)
			env[0] = "PATH=" + filepath.Join("/home/ding/toolchain", goname, "bin") + ":/usr/bin:/bin:/usr/local/bin"
			env = append(env, "GOTOOLCHAIN="+goversion, "DING_GOTOOLCHAIN="+goname)
			slog.Info("building for go", "goname", goname, "goversion", goversion)
			build(env)
		}
		if xgo {
			buildGo("go")
		}
		if goprev {
			buildGo("goprev")
		}
		if gonext {
			buildGo("gonext")
		}
	} else {
		build(environment)
	}

	cleanupTempDestDir()
}
