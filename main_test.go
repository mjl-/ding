package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mjl-/bstore"
	"github.com/mjl-/sherpa"
)

// todo: test SSE events, that they are emitted at the right moments.

var ctxbg = context.Background()

func tcheck(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %s", msg, err)
	}
}

func tcompare(t *testing.T, got, exp any) {
	t.Helper()
	if !reflect.DeepEqual(got, exp) {
		t.Fatalf("got %#v, expected %#v", got, exp)
	}
}

func tneederr(t *testing.T, code string, fn func()) {
	t.Helper()
	defer func() {
		x := recover()
		if x == nil {
			panic(fmt.Sprintf("got no error, expected code %q", code))
		}
		err, ok := x.(*sherpa.Error)
		if !ok {
			panic(fmt.Sprintf("got panic %#v (%T), expected sherpa error code %q", x, x, code))
		} else if err.Code != code {
			panic(fmt.Sprintf("got error code %q, expected %q", err.Code, code))
		}
	}()
	fn()
}

func twaitBuild(t *testing.T, b Build, expStatus BuildStatus) {
	t.Helper()
	api := Ding{}
	for i := 0; i < 100; i++ {
		b = api.Build(ctxbg, config.Password, b.RepoName, b.ID)
		if b.Finish != nil {
			tcompare(t, b.Status, expStatus)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no build result in 10 seconds")
}

func TestMain(m *testing.M) {
	config.Password = "test1234"
	config.DataDir = "testdata/tmp"
	config.ShowSherpaErrors = true
	config.PrintSherpaErrorStack = true
	config.GithubWebhookSecret = "githubsecret"
	config.GiteaWebhookSecret = "giteasecret"
	config.BitbucketWebhookSecret = "bitbucketsecret"
	config.Mail.Enabled = true
	config.Mail.SMTPTLS = true
	config.Mail.SMTPUsername = "test"
	config.Mail.SMTPPassword = "test"
	config.Notify.Email = "ding@example.org"
	config.GoToolchainDir = "testdata/tmp/gotoolchains"

	config.IsolateBuilds.Enabled = os.Getuid() == 0
	if config.IsolateBuilds.Enabled {
		fi, err := os.Stat(".")
		xcheckf(err, "stat dot")
		st := fi.Sys().(*syscall.Stat_t)
		config.IsolateBuilds.DingUID = st.Uid
		config.IsolateBuilds.DingGID = st.Gid

		config.DataDir = "testdata/tmp-root"
	}

	initDingDataDir()
	newSMTPClient = func() smtpClient { return &fakeClient{} }

	os.RemoveAll(config.DataDir)
	os.MkdirAll(config.DataDir, 0777)
	os.MkdirAll(config.GoToolchainDir, 0777)

	privMsg, unprivMsg, privFD, unprivFD := xinitSockets()
	privConn := xunixconn(privFD)
	unprivConn := xunixconn(unprivFD)
	privFD = nil
	unprivFD = nil

	// todo: shut some goroutines down cleanly when done, and wait for completion.
	startJobManager()
	go servePrivileged(gob.NewDecoder(privMsg), gob.NewEncoder(privMsg), privConn)
	go serveUnprivileged(gob.NewDecoder(unprivMsg), gob.NewEncoder(unprivMsg), unprivConn)

	m.Run()

	if database != nil {
		err := database.Close()
		xcheckf(err, "closing database after last test")
	}
}

func testEnv(t *testing.T) {
	if database != nil {
		database.Close()
	}
	os.MkdirAll(config.DataDir, 0700)
	dbpath := fmt.Sprintf("%s/test.%s.db", config.DataDir, strings.ToLower(t.Name()))
	db, err := bstore.Open(context.Background(), dbpath, &bstore.Options{Timeout: 5 * time.Second}, Repo{}, Build{})
	tcheck(t, err, "db open")
	database = db
}
