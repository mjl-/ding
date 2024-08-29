package main

import (
	"compress/gzip"
	"context"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mjl-/bstore"
	"github.com/mjl-/httpinfo"
	"github.com/mjl-/sherpa"
	"github.com/mjl-/sherpadoc"
	"github.com/mjl-/sherpaprom"
)

type job struct {
	repoName string
	lowPrio  bool
	rc       chan struct{}
}

var (
	newJobs      chan job
	finishedJobs chan string // repoName

	rootRequests = make(chan request) // For http-serve, managing comms to privileged process.
)

func servehttp(args []string) {
	serveFlag.Init("serve-http", flag.ExitOnError)
	serveFlag.Usage = func() {
		fmt.Println("usage: ding [flags] serve-http")
		serveFlag.PrintDefaults()
	}
	serveFlag.Parse(args)
	args = serveFlag.Args()
	if len(args) != 0 {
		serveFlag.Usage()
		os.Exit(2)
	}

	msgfile := os.NewFile(3, "msg")
	dec := gob.NewDecoder(msgfile)
	enc := gob.NewEncoder(msgfile)
	err := dec.Decode(&config)
	xcheckf(err, "reading config")

	initDingDataDir()
	if config.Mail.Enabled {
		newSMTPClient = dialSMTPClient
	} else {
		newSMTPClient = func() smtpClient { return &fakeClient{} }
	}

	// Be cautious.
	if config.IsolateBuilds.Enabled && (uint32(os.Getuid()) != config.IsolateBuilds.DingUID || uint32(os.Getgid()) != config.IsolateBuilds.DingGID) {
		slog.Error("not running under expected uid/gid")
		os.Exit(1)
	}

	fdpass := os.NewFile(4, "fdpass")
	unprivConn := xunixconn(fdpass)

	dbpath := path.Join(config.DataDir, "ding.db")
	dbopts := bstore.Options{Timeout: 5 * time.Second}
	database, err = bstore.Open(context.Background(), dbpath, &dbopts, Repo{}, Build{})
	xcheckf(err, "open database")

	var doc sherpadoc.Section
	ff, err := openEmbed("ding.json")
	xcheckf(err, "opening sherpa docs")
	err = json.NewDecoder(ff).Decode(&doc)
	xcheckf(err, "parsing sherpa docs")
	err = ff.Close()
	xcheckf(err, "closing sherpa docs after parsing")

	collector, err := sherpaprom.NewCollector("ding", nil)
	xcheckf(err, "creating sherpa prometheus collector")

	opts := &sherpa.HandlerOpts{
		Collector:           collector,
		AdjustFunctionNames: "none",
	}
	handler, err := sherpa.NewHandler("/ding/", version, Ding{}, &doc, opts)
	xcheckf(err, "making sherpa handler")

	http.Handle("GET /info", httpinfo.NewHandler(httpinfo.CodeVersion{Full: version}, nil))
	http.Handle("GET /metrics", promhttp.Handler())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", serveAsset)
	mux.Handle("GET /ding/", handler)
	mux.Handle("POST /ding/", handler)
	mux.Handle("OPTIONS /ding/", handler)
	mux.HandleFunc("GET /release/", serveRelease)
	mux.HandleFunc("GET /result/", serveResult)
	mux.HandleFunc("GET /dl/", serveDownload)
	mux.HandleFunc("GET /events", serveEvents)

	// Admin is serving the default mux.
	http.HandleFunc("GET /ding.db", func(w http.ResponseWriter, r *http.Request) {
		err := database.Read(r.Context(), func(tx *bstore.Tx) error {
			w.Header().Set("Content-Type", "application/octect-stream")
			_, err := tx.WriteTo(w)
			return err
		})
		if err != nil {
			slog.Debug("writing database dump", "err", err)
		}
	})

	startJobManager()

	staleBuilds, err := bstore.QueryDB[Build](context.Background(), database).FilterFn(func(b Build) bool { return b.Finish == nil }).FilterNotEqual("Status", string(StatusNew)).List()
	xcheckf(err, "listing stale builds in database")
	for _, b := range staleBuilds {
		buildDir := fmt.Sprintf("%s/build/%s/%d/", dingDataDir, b.RepoName, b.ID)
		du := buildDiskUsage(buildDir)

		now := time.Now()
		b.Finish = &now
		b.ErrorMessage = "marked as failed/unfinished at ding startup."
		b.DiskUsage = du
		if err := sherpaCatch(func() { b.Steps = _buildSteps(b) }); err != nil {
			slog.Error("gathering build steps for failed build, ignoring", "err", err)
		}
		err = database.Update(context.Background(), &b)
		xcheckf(err, "marking build as failed/unfinished")
		slog.Info("marked stale build as failed", "builddir", buildDir)
	}

	newBuilds, err := bstore.QueryDB[Build](context.Background(), database).FilterNonzero(Build{Status: StatusNew}).List()
	xcheckf(err, "fetching new builds from database")
	for _, b := range newBuilds {
		repo := Repo{Name: b.RepoName}
		err := database.Get(context.Background(), &repo)
		xcheckf(err, "get repo for new build")

		job := job{
			b.RepoName,
			b.LowPrio,
			make(chan struct{}),
		}
		newJobs <- job
		go func() {
			<-job.rc
			defer func() {
				finishedJobs <- job.repoName
			}()

			buildDir := fmt.Sprintf("%s/build/%s/%d", dingDataDir, b.RepoName, b.ID)
			_doBuild0(context.Background(), repo, b, buildDir)
		}()
	}

	slog.Info("starting ding", "version", version, "addr", *listenAddress, "webhookaddr", *listenWebhookAddress, "adminaddr", *listenAdminAddress)
	if *listenWebhookAddress != "" {
		webhookMux := http.NewServeMux()
		webhookMux.HandleFunc("POST /github/", githubHookHandler)
		webhookMux.HandleFunc("POST /gitea/", giteaHookHandler)
		webhookMux.HandleFunc("POST /bitbucket/", bitbucketHookHandler)
		go func() {
			err := http.ListenAndServe(*listenWebhookAddress, webhookMux)
			slog.Error("listen and serve", "err", err)
			os.Exit(1)
		}()
	}
	if *listenAdminAddress != "" {
		go func() {
			err := http.ListenAndServe(*listenAdminAddress, nil)
			slog.Error("listen and serve", "err", err)
			os.Exit(1)
		}()
	}
	go func() {
		err := http.ListenAndServe(*listenAddress, mux)
		slog.Error("listen and serve", "err", err)
		os.Exit(1)
	}()

	serveUnprivileged(dec, enc, unprivConn)
}

func serveUnprivileged(dec *gob.Decoder, enc *gob.Encoder, unixconn *net.UnixConn) {
	for {
		req := <-rootRequests
		err := enc.Encode(req.msg)
		xcheckf(err, "writing msg to root")

		var r string
		err = dec.Decode(&r)
		xcheckf(err, "reading response from root")

		switch {
		case req.msg.Build != nil:
			if r != "" {
				err = fmt.Errorf("%s", r)
				slog.Error("run failed", "err", err)
				req.buildResponse <- buildResult{err, nil, nil, nil}
				continue
			}

			buf := make([]byte, 1)   // Nothing in there.
			oob := make([]byte, 128) // Expect 3*24 bytes.
			_, oobn, _, _, err := unixconn.ReadMsgUnix(buf, oob)
			xcheckf(err, "receiving fd")
			scms, err := unix.ParseSocketControlMessage(oob[:oobn])
			xcheckf(err, "parsing control message")
			if len(scms) != 1 {
				slog.Error("client: expected 1 SocketControlMessage", "scms", scms)
				os.Exit(1)
			}

			fds, err := unix.ParseUnixRights(&scms[0])
			xcheckf(err, "parse unix rights")
			if len(fds) != 3 {
				slog.Error("wanted 3 fds", "got", len(fds))
				os.Exit(1)
			}

			stdout := os.NewFile(uintptr(fds[0]), fmt.Sprintf("build-%d-stdout", req.msg.Build.BuildID))
			stderr := os.NewFile(uintptr(fds[1]), fmt.Sprintf("build-%d-stderr", req.msg.Build.BuildID))
			status := os.NewFile(uintptr(fds[2]), fmt.Sprintf("build-%d-status", req.msg.Build.BuildID))

			req.buildResponse <- buildResult{nil, stdout, stderr, status}

		default:
			var err error
			if r != "" {
				err = fmt.Errorf("%s", r)
			}
			req.errorResponse <- err
		}
	}
}

func startJobManager() {
	newJobs = make(chan job, 1)
	finishedJobs = make(chan string, 1)

	go func() {
		active := map[string]bool{} // Repo name -> is low prio.
		pending := map[string][]job{}
		pendingLowPrio := []job{}
		lowPrioBusy := false

		kick := func(repoName string) {
			if _, ok := active[repoName]; ok {
				return
			}
			jobs := pending[repoName]
			if len(jobs) == 0 {
				return
			}
			job := jobs[0]
			pending[repoName] = jobs[1:]
			active[repoName] = false
			job.rc <- struct{}{}
		}

		kickLowPrio := func() {
			if lowPrioBusy {
				return
			}
			for i, job := range pendingLowPrio {
				_, ok := active[job.repoName]
				if len(pending[job.repoName]) == 0 && !ok {
					lowPrioBusy = true
					pendingLowPrio = append(pendingLowPrio[:i], pendingLowPrio[i+1:]...)
					active[job.repoName] = true
					job.rc <- struct{}{}
					return
				}
			}
		}

		for {
			select {
			case job := <-newJobs:
				if job.lowPrio {
					pendingLowPrio = append(pendingLowPrio, job)
					kickLowPrio()
				} else {
					pending[job.repoName] = append(pending[job.repoName], job)
					kick(job.repoName)
				}

			case repoName := <-finishedJobs:
				lowPrio := active[repoName]
				delete(active, repoName)
				kick(repoName)
				if lowPrio {
					lowPrioBusy = false
					kickLowPrio()
				}
			}
		}
	}()
}

type readSeekStatCloser interface {
	Stat() (fs.FileInfo, error)
	io.ReadSeekCloser
}

func openEmbed(path string) (readSeekStatCloser, error) {
	f, err := os.Open(path)
	if err == nil {
		return f, nil
	}
	ef, err := embedFS.Open(path)
	if err != nil {
		return nil, err
	}
	r, ok := ef.(readSeekStatCloser)
	if !ok {
		r.Close()
		return nil, fmt.Errorf("embedded file not a readseekcloser")
	}
	return r, nil
}

func serveAsset(w http.ResponseWriter, r *http.Request) {
	var path string
	switch r.URL.Path {
	case "/":
		path = "ding.html"
	case "/ding.js", "/favicon.ico":
		path = r.URL.Path[1:]
	default:
		http.NotFound(w, r)
		return
	}

	f, err := openEmbed(path)
	if err != nil {
		http.Error(w, "500 - internal server error - "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		slog.Error("serving asset", "path", r.URL.Path, "err", err)
		http.Error(w, "500 - internal server error - "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, haveCacheBuster := r.URL.Query()["v"]
	cache := "no-cache, max-age=0"
	if haveCacheBuster {
		cache = fmt.Sprintf("public, max-age=%d", 31*24*3600)
	}
	w.Header().Set("Cache-Control", cache)

	http.ServeContent(w, r, r.URL.Path, info.ModTime(), f)
}

func hasBadElems(elems []string) bool {
	for _, e := range elems {
		switch e {
		case "", ".", "..":
			return true
		}
	}
	return false
}

func serveRelease(w http.ResponseWriter, r *http.Request) {
	t := strings.Split(r.URL.Path[1:], "/")
	if len(t) != 4 || hasBadElems(t[1:]) {
		http.NotFound(w, r)
		return
	}

	name := t[3]
	path := fmt.Sprintf("%s/release/%s/%s/%s.gz", dingDataDir, t[1], t[2], name)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if acceptsGzip(r.Header.Get("Accept-Encoding")) {
		w.Header().Set("Content-Encoding", "gzip")
		io.Copy(w, f) // Nothing to do for errors.
	} else {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			slog.Error("release: reading gzip file", "path", path, "err", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		io.Copy(w, gzr) // Nothing to do for errors.
	}
}

func acceptsGzip(s string) bool {
	t := strings.Split(s, ",")
	for _, e := range t {
		e = strings.TrimSpace(e)
		tt := strings.Split(e, ";")
		if len(tt) > 1 && t[1] == "q=0" {
			continue
		}
		if tt[0] == "gzip" {
			return true
		}
	}
	return false
}

func serveResult(w http.ResponseWriter, r *http.Request) {
	t := strings.Split(r.URL.Path[1:], "/")
	if len(t) != 4 || hasBadElems(t[1:]) {
		http.NotFound(w, r)
		return
	}
	repoName := t[1]
	buildID, err := strconv.Atoi(t[2])
	if err != nil || repoName == "" || buildID == 0 {
		http.NotFound(w, r)
		return
	}
	filename := t[3]

	var p string
	err = database.Read(r.Context(), func(tx *bstore.Tx) error {
		repo := Repo{Name: repoName}
		if err := tx.Get(&repo); err != nil {
			return err
		}
		b, err := bstore.QueryTx[Build](tx).FilterNonzero(Build{ID: int32(buildID), RepoName: repoName}).Get()
		if err != nil {
			return err
		}
		suffix := "/" + filename
		for _, res := range b.Results {
			if res.Filename == filename || strings.HasSuffix(res.Filename, suffix) {
				p = fmt.Sprintf("%s/build/%s/%d/checkout/%s/%s", dingDataDir, repoName, b.ID, repo.CheckoutPath, res.Filename)
				break
			}
		}
		return nil
	})
	if err == bstore.ErrAbsent || err == nil && p == "" {
		http.NotFound(w, r)
	} else if err != nil {
		slog.Error("fetching build results from database", "err", err)
		http.Error(w, "500 internal error", http.StatusInternalServerError)
	} else {
		http.ServeFile(w, r, p)
	}
}
