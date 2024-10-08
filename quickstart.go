package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"syscall"

	"github.com/mjl-/bstore"
	"github.com/mjl-/sconf"
)

func quickstart(args []string) {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: ding quickstart")
		os.Exit(2)
	}

	if os.Getuid() != 0 {
		fmt.Fprintln(os.Stderr, "quickstart must be run as root")
		os.Exit(2)
	}
	workdir, err := os.Getwd()
	xcheckf(err, "get workdir")

	wdfi, err := os.Stat(workdir)
	xcheckf(err, "stat workdir")
	wdst, ok := wdfi.Sys().(*syscall.Stat_t)
	if !ok {
		xcheckf(errors.New("stat not a syscall.Stat_t"), "stat workdir")
	}

	u, err := user.Lookup("ding")
	if err != nil {
		if runtime.GOOS == "freebsd" {
			fmt.Fprintf(os.Stderr, "looking up user ding: %v\nHint: pw useradd ding -d %s\n", err, workdir)
		} else {
			fmt.Fprintf(os.Stderr, "looking up user ding: %v\nHint: useradd -d %s ding\n", err, workdir)
		}
		os.Exit(2)
	}
	uid, err := strconv.ParseInt(u.Uid, 10, 64)
	xcheckf(err, "parsing ding uid")
	gid, err := strconv.ParseInt(u.Gid, 10, 64)
	xcheckf(err, "parsing ding gid")

	c := Config{
		ShowSherpaErrors:      true,
		PrintSherpaErrorStack: true,
		Password:              genSecret(),
		DataDir:               "data",
		GoToolchainDir:        "toolchains",
		BaseURL:               "http://localhost:6084",
	}
	c.IsolateBuilds.Enabled = true
	c.IsolateBuilds.UIDStart = 10000
	c.IsolateBuilds.UIDEnd = 20000
	c.IsolateBuilds.DingUID = uint32(uid)
	c.IsolateBuilds.DingGID = uint32(gid)
	cf, err := os.OpenFile("ding.conf", os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0640)
	if err != nil && os.IsExist(err) {
		fmt.Fprintf(os.Stderr, `error: ding.conf already exists.
If you're retrying after a previous error, make sure to remove the previously
generated files and directories (ding.conf, data, toolchains and optionally
ding.service).
`)
		os.Exit(2)
	}
	xcheckf(err, "create ding.conf")
	err = sconf.WriteDocs(cf, c)
	xcheckf(err, "writing config file")
	err = cf.Close()
	xcheckf(err, "close config file")

	err = os.Mkdir(c.DataDir, 0770)
	xcheckf(err, "making data dir")

	err = os.Mkdir(c.GoToolchainDir, 0750)
	xcheckf(err, "making toolchain dir")

	writeRC := func(name string) {
		sf, err := fsys.Open(name)
		xcheckf(err, "open rc file")
		defer sf.Close()
		rc, err := io.ReadAll(sf)
		xcheckf(err, "read rc file")
		rc = bytes.ReplaceAll(rc, []byte("/home/service/ding"), []byte(workdir))
		err = os.WriteFile("ding.rc", rc, 0755)
		xcheckf(err, "writing rc file")

		err = os.Chown("ding.rc", 0, 0)
		xcheckf(err, "chown ding.rc")
		err = os.Chmod("ding.rc", 0755)
		xcheckf(err, "chmod ding.rc")
	}

	switch runtime.GOOS {
	case "linux":
		sf, err := fsys.Open("ding.service")
		xcheckf(err, "open service file")
		defer sf.Close()
		service, err := io.ReadAll(sf)
		xcheckf(err, "read service file")
		service = bytes.ReplaceAll(service, []byte("/home/service/ding"), []byte(workdir))
		err = os.WriteFile("ding.service", service, 0644)
		xcheckf(err, "writing service file")

		err = os.Chown("ding.service", int(wdst.Uid), int(wdst.Gid))
		xcheckf(err, "chown ding.service")
		err = os.Chmod("ding.service", 0644)
		xcheckf(err, "chmod ding.service")

	case "openbsd":
		writeRC("ding.openbsd.rc")

	case "freebsd":
		writeRC("ding.freebsd.rc")
	}

	db, err := bstore.Open(context.Background(), "data/ding.db", nil, dbtypes...)
	xcheckf(err, "open db file")
	settings := Settings{
		ID:                       1, // singleton
		Environment:              []string{"PATH=/usr/bin:/bin:/usr/local/bin"},
		AutomaticGoToolchains:    true,
		GoToolchainWebhookSecret: "Bearer " + genSecret(),
	}
	if runtime.GOOS == "linux" {
		settings.RunPrefix = []string{"nice", "ionice", "-c3", "timeout", "600s"}
	} else {
		settings.RunPrefix = []string{"nice", "timeout", "600s"}
	}

	err = db.Insert(context.Background(), &settings)
	xcheckf(err, "inserting settings in database")
	err = db.Close()
	xcheckf(err, "closing database")
	db = nil

	// Fix permissions of workdir and files.
	err = os.Chown(workdir, int(wdst.Uid), int(gid))
	xcheckf(err, "chown workdir to ding gid")
	err = os.Chmod(workdir, 02750)
	xcheckf(err, "chmod workdir")

	err = os.Chown(c.DataDir, int(wdst.Uid), int(gid))
	xcheckf(err, "chown datadir to ding gid")
	err = os.Chmod(c.DataDir, 02770)
	xcheckf(err, "chmod datadir")

	err = os.Chown(c.DataDir+"/ding.db", int(wdst.Uid), int(gid))
	xcheckf(err, "chown ding.db to ding gid")
	err = os.Chmod(c.DataDir+"/ding.db", 0660)
	xcheckf(err, "chmod ding.db")

	err = os.Chown(c.GoToolchainDir, int(wdst.Uid), int(gid))
	xcheckf(err, "chown toolchaindir to ding gid")
	err = os.Chmod(c.GoToolchainDir, 02750)
	xcheckf(err, "chmod toolchaindir")

	err = os.Chown("ding.conf", int(wdst.Uid), int(gid))
	xcheckf(err, "chown ding.conf to ding gid")
	err = os.Chmod("ding.conf", 0640)
	xcheckf(err, "chmod ding.conf")

	err = os.Chown("ding", int(wdst.Uid), int(gid))
	xcheckf(err, "chown ding to ding gid")
	err = os.Chmod("ding", 0750)
	xcheckf(err, "chmod ding")

	fmt.Printf(`Wrote config file to ding.conf. Please review its contents, and optionally
configure SMTP settings for sending notification emails for broken builds and
base URL.

Generated password: %s (for logging into web interface)
`, c.Password)
	switch runtime.GOOS {
	case "linux":
		fmt.Printf(`
A systemd service file has been written to ding.service. To install as service and start:

	systemctl enable $PWD/ding.service
	systemctl start ding.service
	journalctl -f -u ding.service # See logs
`)
	case "openbsd":
		fmt.Printf(`
An rc.d script has been written to ding.rc. To install as service and start:

	mv ding.rc /etc/rc.d/ding
	rcctl enable ding
	rcctl start ding
`)
	case "freebsd":
		fmt.Printf(`
An rc.d script has been written to ding.rc. To install as service and start:

	mv ding.rc /etc/rc.d/ding
	sysrc ding_enable="YES"
	service ding start
`)
	}
	fmt.Printf(`
You can start ding manually by running:

	umask 027
	./ding -loglevel debug serve -listen localhost:6084 -listenwebhook localhost:6085 -listenadmin localhost:6086 ding.conf

After starting, ding will serve on:

- http://localhost:6084, the web interface, check the settings and toolchains page
- http://localhost:6085, web hooks, to trigger builds from version control systems
- http://localhost:6086, admin endpoint, for prometheus metrics and database dumps

You may want to configure a reverse proxy, or change the IPs ding listens to
internal VPN IPs.

See the output of "ding help" for instructions on configuring backups.
`)
}
