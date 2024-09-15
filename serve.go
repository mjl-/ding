package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/mjl-/sconf"
)

var (
	dingDataDir          string // config.DataDir combined with working directory of ding.
	serveFlag            = flag.NewFlagSet("serve", flag.ExitOnError)
	listenAddress        = serveFlag.String("listen", "localhost:6084", "address to listen on")
	listenWebhookAddress = serveFlag.String("listenwebhook", "localhost:6085", "address to listen on for webhooks, like from github; set empty for no listening")
	listenAdminAddress   = serveFlag.String("listenadmin", "localhost:6086", "address to listen on for monitoring endpoints like prometheus /metrics and /info")
)

func serve(args []string) {
	origArgs := args
	serveFlag.Init("serve", flag.ExitOnError)
	serveFlag.Usage = func() {
		fmt.Println("usage: ding [flags] serve ding.conf")
		serveFlag.PrintDefaults()
	}
	serveFlag.Parse(args)
	args = serveFlag.Args()
	if len(args) != 1 {
		serveFlag.Usage()
		os.Exit(2)
	}

	err := sconf.ParseFile(args[0], &config)
	xcheckf(err, "parsing config file")

	initDingDataDir()

	if config.IsolateBuilds.Enabled {
		if os.Getuid() != 0 {
			slog.Error(`must run as root when isolateBuilds is enabled`)
			os.Exit(1)
		}
		if syscall.Umask(027) != 027 {
			slog.Error("must run with umask 027 with isolateBuilds enabled")
			os.Exit(1)
		}
		info, err := os.Stat(dingDataDir)
		xcheckf(err, "stat data dir")
		sysinfo := info.Sys()
		if sysinfo == nil {
			slog.Error("cannot determine owner of data dir", "datadir", dingDataDir)
			os.Exit(1)
		}
		st, ok := sysinfo.(*syscall.Stat_t)
		if !ok {
			slog.Error("underlying fileinfo for data dir", "datadir", dingDataDir, "systype", fmt.Sprintf("%T", sysinfo))
			os.Exit(1)
		}
		if info.Mode()&077 != 070 || st.Gid != config.IsolateBuilds.DingGID {
			slog.Error("data dir must have permissions g=rwx,o= and ding gid", "datadir", dingDataDir, "expectgid", config.IsolateBuilds.DingGID, "gotperm", info.Mode()&os.ModePerm, "gotgit", st.Gid)
			os.Exit(1)
		}
	} else {
		if os.Getuid() == 0 {
			slog.Error(`must not run as root when isolateBuilds is disabled`)
			os.Exit(1)
		}
	}

	privMsg, unprivMsg, privFD, unprivFD := xinitSockets()
	privConn := xunixconn(privFD)
	privFD = nil

	argv := append([]string{os.Args[0], "-loglevel=" + loglevel.Level().String(), "serve-http"}, origArgs[:len(origArgs)-1]...)
	attr := &os.ProcAttr{
		Files: []*os.File{
			os.Stdin,
			os.Stdout,
			os.Stderr,
			unprivMsg,
			unprivFD,
		},
	}
	if config.IsolateBuilds.Enabled {
		attr.Sys = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid:    config.IsolateBuilds.DingUID,
				Gid:    config.IsolateBuilds.DingGID,
				Groups: []uint32{},
			},
		}
	}
	proc, err := os.StartProcess(argv[0], argv, attr)
	xcheckf(err, "starting http process")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)
	go func() {
		<-sig
		err := proc.Signal(syscall.SIGTERM)
		if err != nil {
			slog.Error("sending sigterm to serve-http process", "err", err)
		}
		os.Exit(1)
	}()

	xcheckf(unprivMsg.Close(), "closing unpriv msg file")
	xcheckf(unprivFD.Close(), "closing unpriv fd file")
	unprivMsg = nil
	unprivFD = nil

	dec := gob.NewDecoder(privMsg)
	enc := gob.NewEncoder(privMsg)
	err = enc.Encode(&config)
	xcheckf(err, "writing config to httpserver")
	servePrivileged(dec, enc, privConn)
}

func xinitSockets() (privMsg, unprivMsg, privFD, unprivFD *os.File) {
	proto := 0
	// We exchange gob messages with unprivileged httpserver over msgpair.
	msgpair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, proto)
	xcheckf(err, "creating socketpair")

	// And we send file descriptors over fdpair to unprivileged httpserver after
	// kicking off a build under a unique uid.
	fdpair, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, proto)
	xcheckf(err, "creating socketpair")

	privMsg = os.NewFile(uintptr(msgpair[0]), "privmsg")
	unprivMsg = os.NewFile(uintptr(msgpair[1]), "unprivmsg")
	privFD = os.NewFile(uintptr(fdpair[0]), "privfd")
	unprivFD = os.NewFile(uintptr(fdpair[1]), "unprivfd")

	return
}

func xunixconn(f *os.File) *net.UnixConn {
	fc, err := net.FileConn(f)
	xcheckf(err, "fileconn")
	uc, ok := fc.(*net.UnixConn)
	if !ok {
		slog.Error("file not a unixconn")
		os.Exit(1)
	}
	err = f.Close()
	xcheckf(err, "closing unix file")
	return uc
}

func servePrivileged(dec *gob.Decoder, enc *gob.Encoder, unixconn *net.UnixConn) {
	for {
		var msg msg
		err := dec.Decode(&msg)
		xcheckf(err, "decoding msg")

		switch {
		case msg.Build != nil:
			err = doMsgBuild(msg.Build, enc, unixconn)
		case msg.Chown != nil:
			err = doMsgChown(msg.Chown, enc)
		case msg.RemoveBuilddir != nil:
			err = doMsgRemoveBuilddir(msg.RemoveBuilddir, enc)
		case msg.RemoveRepo != nil:
			err = doMsgRemoveRepo(msg.RemoveRepo, enc)
		case msg.RemoveSharedHome != nil:
			err = doMsgRemoveSharedHome(msg.RemoveSharedHome, enc)
		case msg.CancelCommand != nil:
			err = doMsgCancelCommand(msg.CancelCommand, enc)
		case msg.InstallGoToolchain != nil:
			err = installGoToolchain(msg.InstallGoToolchain.File, msg.InstallGoToolchain.Shortname)
		case msg.RemoveGoToolchain != nil:
			err = removeGoToolchain(msg.RemoveGoToolchain.Goversion)
		case msg.ActivateGoToolchain != nil:
			err = activateGoToolchain(msg.ActivateGoToolchain.Goversion, msg.ActivateGoToolchain.Shortname)
		case msg.AutomaticGoToolchain != nil:
			// todo: replace horrible hack of setting a specific value as error to communicate we've updated the toolchains...
			var updated bool
			updated, err = automaticGoToolchain()
			if err == nil && updated {
				err = errors.New("updated")
			}
		case msg.LogLevelSet != nil:
			olevel := loglevel.Level()
			loglevel.Set(msg.LogLevelSet.LogLevel)
			slog.Warn("log level changed", "oldlevel", olevel, "newlevel", loglevel.Level())
		default:
			slog.Error("no field set in msg", "msg", msg)
			os.Exit(2)
		}

		err = enc.Encode(errstr(err))
		xcheckf(err, "writing response")
	}
}

func errstr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

var errBadParams = errors.New("bad parameters")

func doMsgChown(msg *msgChown, enc *gob.Encoder) error {
	if !config.IsolateBuilds.Enabled {
		return nil
	}

	buildDir := fmt.Sprintf("%s/build/%s/%d", dingDataDir, msg.RepoName, msg.BuildID)

	slog.Debug("changing ownership", "builddir", buildDir)

	homeDir := fmt.Sprintf("%s/home", buildDir)
	if msg.SharedHome {
		homeDir = fmt.Sprintf("%s/home/%s", dingDataDir, msg.RepoName)
	}
	if path.Clean(buildDir) != buildDir || path.Clean(homeDir) != homeDir {
		return errBadParams
	}

	chown := func(path string) error {
		return filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Don't change symlinks, we would be modifying whatever they point to!
			if (info.Mode() & os.ModeSymlink) != 0 {
				return nil
			}
			return os.Chown(path, int(msg.UID), int(config.IsolateBuilds.DingGID))
		})
	}

	err := chown(homeDir)
	if err == nil {
		err = chown(buildDir + "/checkout")
	}
	if err == nil {
		err = chown(buildDir + "/dl")
	}
	return err
}

func ensureWritable(dir string) {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Don't change symlinks, we would be modifying whatever they point to!
		if (info.Mode() & os.ModeSymlink) != 0 {
			return nil
		}
		if (info.Mode() & 0200) == 0 {
			if err := os.Chmod(path, info.Mode()|0200); err != nil {
				slog.Error("making path writable before removing", "path", path, "err", err)
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("walking dir to ensure files are writable, for removal", "dir", dir, "err", err)
	}
}

func doMsgRemoveBuilddir(msg *msgRemoveBuilddir, enc *gob.Encoder) error {
	p := fmt.Sprintf("%s/build/%s/%d", dingDataDir, msg.RepoName, msg.BuildID)
	slog.Debug("removing build dir", "builddir", p)
	if path.Clean(p) != p {
		return errBadParams
	}
	ensureWritable(p)
	return os.RemoveAll(p)
}

func doMsgRemoveRepo(msg *msgRemoveRepo, enc *gob.Encoder) error {
	homeDir := fmt.Sprintf("%s/home/%s", dingDataDir, msg.RepoName)
	repoDir := fmt.Sprintf("%s/build/%s", dingDataDir, msg.RepoName)

	slog.Debug("removing repository", "repo", msg.RepoName, "homedir", homeDir, "repodir", repoDir)

	if path.Clean(homeDir) != homeDir || path.Clean(repoDir) != repoDir {
		return errBadParams
	}

	ensureWritable(homeDir)
	err := os.RemoveAll(homeDir)
	if err != nil && os.IsNotExist(err) {
		err = nil
	}
	ensureWritable(repoDir)
	err2 := os.RemoveAll(repoDir)
	if err == nil {
		err = err2
	}
	return err
}

func doMsgRemoveSharedHome(msg *msgRemoveSharedHome, enc *gob.Encoder) error {
	homeDir := fmt.Sprintf("%s/home/%s", dingDataDir, msg.RepoName)

	slog.Debug("removing shared homedir", "repo", msg.RepoName, "homedir", homeDir)

	if path.Clean(homeDir) != homeDir {
		return errBadParams
	}
	ensureWritable(homeDir)
	err := os.RemoveAll(homeDir)
	if err != nil && os.IsNotExist(err) {
		err = nil
	}
	return err
}

func doMsgCancelCommand(msg *msgCancelCommand, enc *gob.Encoder) error {
	buildIDCommandCancel(msg.BuildID)
	return nil
}

func bwrapCmd(nonet bool, homeDir, buildDir, checkoutPath, toolchainDir string) []string {
	argv := []string{"bwrap", "--die-with-parent"}
	if nonet {
		argv = append(argv, "--unshare-all")
	} else {
		argv = append(argv, "--unshare-user-try", "--unshare-ipc", "--unshare-pid", "--unshare-uts", "--unshare-cgroup-try")
	}
	argv = append(argv,
		"--hostname", "ding",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
		"--proc", "/proc",
		"--ro-bind", "/etc", "/etc",
		"--ro-bind", "/bin", "/bin",
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind-try", "/lib32", "/lib32",
		"--ro-bind-try", "/lib64", "/lib64",
		"--bind", homeDir, "/home/ding",
		"--bind", buildDir, "/home/ding/build",
	)
	if toolchainDir != "" {
		argv = append(argv, "--bind", toolchainDir, "/home/ding/toolchain")
	}
	argv = append(argv, "--chdir", "/home/ding/build/checkout/"+checkoutPath)
	return argv
}

func doMsgBuild(msg *msgBuild, enc *gob.Encoder, unixconn *net.UnixConn) error {
	buildCommand := buildIDCommandRegister(msg.BuildID)
	needCancel := true
	defer func() {
		if needCancel {
			buildIDCommandCancel(msg.BuildID)
		}
	}()

	if config.IsolateBuilds.Enabled && (msg.UID < config.IsolateBuilds.UIDStart || msg.UID >= config.IsolateBuilds.UIDEnd) {
		return errBadParams
	}

	buildDir := fmt.Sprintf("%s/build/%s/%d", dingDataDir, msg.RepoName, msg.BuildID)
	workDir := fmt.Sprintf("%s/checkout/%s", buildDir, msg.CheckoutPath)
	if path.Clean(buildDir) != buildDir || path.Clean(workDir) != workDir {
		return errBadParams
	}

	var buildEnvs [][]string
	var zt GoToolchains
	if msg.GoToolchains == zt {
		buildEnvs = append(buildEnvs, msg.Env)
	} else {
		addBuildEnv := func(goversion, goname string) {
			if goversion == "" {
				return
			}
			slog.Debug("will be building for go toolchain", "repo", msg.RepoName, "goname", goname, "goversion", goversion)
			env := append([]string{}, msg.Env...)
			gotoolchainpath := msg.ToolchainDir
			if msg.Bubblewrap {
				gotoolchainpath = "/home/ding/toolchain"
			}
			gotoolchainpath = path.Join(gotoolchainpath, goname, "bin")
			var have bool
			for i, e := range env {
				if strings.HasPrefix(e, "PATH=") {
					env[i] = "PATH=" + gotoolchainpath + ":" + strings.TrimPrefix(e, "PATH=")
					have = true
					break
				}
			}
			if !have {
				env = append(env, "PATH="+gotoolchainpath+":/usr/bin:/bin:/usr/local/bin")
			}
			env = append(env, "GOTOOLCHAIN="+goversion)
			env = append(env, "DING_GOTOOLCHAIN="+goname)
			if msg.NewGoToolchain {
				env = append(env, "DING_NEWGOTOOLCHAIN=yes")
			}
			buildEnvs = append(buildEnvs, env)
		}
		addBuildEnv(msg.GoToolchains.Go, "go")
		addBuildEnv(msg.GoToolchains.GoPrev, "goprev")
		addBuildEnv(msg.GoToolchains.GoNext, "gonext")
		if len(buildEnvs) == 0 {
			return fmt.Errorf("internal error: no go toolchain build envs")
		}
	}

	outr, outw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %v", err)
	}

	errr, errw, err := os.Pipe()
	if err != nil {
		outr.Close()
		outw.Close()
		return fmt.Errorf("create stderr pipe: %v", err)
	}

	statusr, statusw, err := os.Pipe()
	if err != nil {
		outr.Close()
		outw.Close()
		errr.Close()
		errw.Close()
		return fmt.Errorf("create status pipe: %v", err)
	}

	buf := []byte{1}
	oob := unix.UnixRights(int(outr.Fd()), int(errr.Fd()), int(statusr.Fd()))
	_, _, err = unixconn.WriteMsgUnix(buf, oob, nil)
	xcheckf(err, "sending fds from root to http")
	outr.Close()
	errr.Close()
	statusr.Close()
	needCancel = false

	argv := []string{}
	envBuildDir := buildDir
	if msg.Bubblewrap {
		envBuildDir = "/home/ding/build"
		argv = bwrapCmd(msg.BubblewrapNoNet, msg.HomeDir, buildDir, msg.CheckoutPath, msg.ToolchainDir)
	}
	argv = append(argv, msg.RunPrefix...)
	argv = append(argv, envBuildDir+"/scripts/build.sh")

	// todo: we are now running each build command in the build step, with one big output. should split them up and show their results separately too. including their own test coverage.

	go func() {
		defer outw.Close()
		defer errw.Close()
		defer statusw.Close()
		defer buildIDCommandCancel(msg.BuildID)

		var err error
		for _, env := range buildEnvs {
			cmd := exec.CommandContext(buildCommand.ctx, argv[0], argv[1:]...)
			cmd.Dir = workDir
			cmd.Env = env
			cmd.Stdout = outw
			cmd.Stderr = errw
			uidgid := ""
			if config.IsolateBuilds.Enabled {
				uidgid = fmt.Sprintf("%d/%d", msg.UID, config.IsolateBuilds.DingGID)
				cmd.SysProcAttr = &syscall.SysProcAttr{
					Credential: &syscall.Credential{
						Uid:    msg.UID,
						Gid:    config.IsolateBuilds.DingGID,
						Groups: []uint32{},
					},
				}
			}

			slog.Debug("running build command", "repo", msg.RepoName, "buildid", msg.BuildID, "builddir", buildDir, "workdir", workDir, "cmd", argv, "uidgid", uidgid, "env", env)

			if err = cmd.Start(); err != nil {
				slog.Error("starting command", "err", err)
				break
			}
			if err = cmd.Wait(); err != nil {
				slog.Error("command result", "err", err)
				break
			}
		}
		err = gob.NewEncoder(statusw).Encode(errstr(err))
		xcheckf(err, "writing status to http-serve")
	}()

	return nil
}
