package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
	log.SetFlags(0)
	log.SetPrefix("serve: ")
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
			log.Fatalln(`must run as root when isolateBuilds is enabled`)
		}
		if syscall.Umask(027) != 027 {
			log.Fatalln("must run with umask 027 with isolateBuilds enabled")
		}
		info, err := os.Stat(dingDataDir)
		xcheckf(err, "stat data dir")
		sysinfo := info.Sys()
		if sysinfo == nil {
			log.Fatalf("cannot determine owner of data dir %q", dingDataDir)
		}
		st, ok := sysinfo.(*syscall.Stat_t)
		if !ok {
			log.Fatalf("underlying fileinfo for data dir %q: sys is a %T", dingDataDir, sysinfo)
		}
		if info.Mode()&077 != 070 || st.Gid != config.IsolateBuilds.DingGID {
			log.Fatalf("data dir %q must have permissions g=rwx,o= and ding gid %d, but has permissions %#o and gid %d", dingDataDir, config.IsolateBuilds.DingGID, info.Mode()&os.ModePerm, st.Gid)
		}
	} else {
		if os.Getuid() == 0 {
			log.Fatalln(`must not run as root when isolateBuilds is disabled`)
		}
	}

	privMsg, unprivMsg, privFD, unprivFD := xinitSockets()
	privConn := xunixconn(privFD)
	privFD = nil

	argv := append([]string{os.Args[0], "serve-http"}, os.Args[2:len(os.Args)-1]...)
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
	_, err = os.StartProcess(argv[0], argv, attr)
	xcheckf(err, "starting http process")

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
		log.Fatalln("file not a unixconn")
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
		default:
			log.Fatalf("no field set in msg %v", msg)
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
				log.Printf("making path writable before removing, %q: %v", path, err)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("walking dir %q to ensure files are writable, for removal: %v", dir, err)
	}
}

func doMsgRemoveBuilddir(msg *msgRemoveBuilddir, enc *gob.Encoder) error {
	p := fmt.Sprintf("%s/build/%s/%d", dingDataDir, msg.RepoName, msg.BuildID)
	if path.Clean(p) != p {
		return errBadParams
	}
	ensureWritable(p)
	return os.RemoveAll(p)
}

func doMsgRemoveRepo(msg *msgRemoveRepo, enc *gob.Encoder) error {
	homeDir := fmt.Sprintf("%s/home/%s", dingDataDir, msg.RepoName)
	repoDir := fmt.Sprintf("%s/build/%s", dingDataDir, msg.RepoName)
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

func doMsgBuild(msg *msgBuild, enc *gob.Encoder, unixconn *net.UnixConn) error {
	buildCommand := buildIDCommandRegister(msg.BuildID)
	needCancel := true
	defer func() {
		if needCancel {
			buildIDCommandCancel(msg.BuildID)
		}
	}()

	outr, outw, err := os.Pipe()
	xcheckf(err, "create stdout pipe")
	defer outr.Close()
	defer outw.Close()

	errr, errw, err := os.Pipe()
	xcheckf(err, "create stderr pipe")
	defer errr.Close()
	defer errw.Close()

	buildDir := fmt.Sprintf("%s/build/%s/%d", dingDataDir, msg.RepoName, msg.BuildID)
	workDir := fmt.Sprintf("%s/checkout/%s", buildDir, msg.CheckoutPath)
	if path.Clean(buildDir) != buildDir || path.Clean(workDir) != workDir {
		return errBadParams
	}

	cmd := exec.CommandContext(buildCommand.ctx, buildDir+"/scripts/build.sh")
	cmd.Dir = workDir
	cmd.Env = msg.Env
	cmd.Stdout = outw
	cmd.Stderr = errw
	if config.IsolateBuilds.Enabled {
		if msg.UID < config.IsolateBuilds.UIDStart || msg.UID >= config.IsolateBuilds.UIDEnd {
			return errBadParams
		}
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid:    msg.UID,
				Gid:    config.IsolateBuilds.DingGID,
				Groups: []uint32{},
			},
		}
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	statusr, statusw, err := os.Pipe()
	xcheckf(err, "create status pipe")

	buf := []byte{1}
	oob := unix.UnixRights(int(outr.Fd()), int(errr.Fd()), int(statusr.Fd()))
	_, _, err = unixconn.WriteMsgUnix(buf, oob, nil)
	xcheckf(err, "sending fds from root to http")
	statusr.Close()

	needCancel = false

	go func() {
		defer statusw.Close()
		defer buildIDCommandCancel(msg.BuildID)

		err := cmd.Wait()
		err = gob.NewEncoder(statusw).Encode(errstr(err))
		xcheckf(err, "writing status to http-serve")
	}()

	return nil
}
