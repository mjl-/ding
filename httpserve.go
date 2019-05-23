package main

import (
	"compress/gzip"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mjl-/httpinfo"
	"github.com/mjl-/sherpa"
	"github.com/mjl-/sherpadoc"
	"github.com/mjl-/sherpaprom"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/sys/unix"
)

type job struct {
	repoName string
	rc       chan struct{}
}

var (
	newJobs      chan job
	finishedJobs chan string // repoName
)

func servehttp(args []string) {
	log.SetFlags(0)
	log.SetPrefix("http-serve: ")
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
	err := dec.Decode(&config)
	check(err, "reading config")

	initDingDataDir()

	// be cautious
	if config.IsolateBuilds.Enabled && (uint32(os.Getuid()) != config.IsolateBuilds.DingUID || uint32(os.Getgid()) != config.IsolateBuilds.DingGID) {
		log.Fatalln("not running under expected uid/gid")
	}

	fdpass := os.NewFile(4, "fdpass")
	fileconn, err := net.FileConn(fdpass)
	check(err, "making fileconn from fd")
	check(fdpass.Close(), "closing original fdpass")
	unixconn, ok := fileconn.(*net.UnixConn)
	if !ok {
		log.Fatalln("fd 4 not a unixconn")
	}

	rootRequests = make(chan request)

	database, err = sql.Open("postgres", config.Database)
	check(err, "opening database connection")
	var dbVersion int
	err = database.QueryRow("select max(version) from schema_upgrades").Scan(&dbVersion)
	check(err, "fetching database schema version")
	if dbVersion != databaseVersion {
		log.Fatalf("bad database schema version, expected %d, saw %d", databaseVersion, dbVersion)
	}

	// so http package returns these known mimetypes
	mime.AddExtensionType(".woff2", "font/woff2")
	mime.AddExtensionType(".ttf", "font/ttf")
	mime.AddExtensionType(".otf", "font/otf")

	var doc sherpadoc.Section
	ff, err := httpFS.Open("/ding.json")
	check(err, "opening sherpa docs")
	err = json.NewDecoder(ff).Decode(&doc)
	check(err, "parsing sherpa docs")
	err = ff.Close()
	check(err, "closing sherpa docs after parsing")

	collector, err := sherpaprom.NewCollector("ding", nil)
	check(err, "creating sherpa prometheus collector")

	opts := &sherpa.HandlerOpts{
		Collector: collector,
	}
	handler, err := sherpa.NewHandler("/ding/", version, Ding{}, &doc, opts)
	check(err, "making sherpa handler")

	// Since we set the version variables with ldflags -X, we cannot read them in the vars section.
	// So we combine them into a CodeVersion during init, and add the handler while we're at it.
	info := httpinfo.CodeVersion{
		CommitHash: vcsCommitHash,
		Tag:        vcsTag,
		Branch:     vcsBranch,
		Full:       version,
	}
	http.Handle("/info", httpinfo.NewHandler(info, nil))

	http.HandleFunc("/", serveAsset)
	http.Handle("/ding/", handler)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/release/", serveRelease)
	http.HandleFunc("/result/", serveResult)
	http.HandleFunc("/download/", serveDownload) // Old
	http.HandleFunc("/dl/", serveDownload)       // New
	http.HandleFunc("/events", serveEvents)

	go eventMux()

	newJobs = make(chan job, 1)
	finishedJobs = make(chan string, 1)
	go func() {
		active := map[string]struct{}{}
		pending := map[string][]job{}

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
			active[repoName] = struct{}{}
			job.rc <- struct{}{}
		}

		for {
			select {
			case job := <-newJobs:
				pending[job.repoName] = append(pending[job.repoName], job)
				kick(job.repoName)

			case repoName := <-finishedJobs:
				delete(active, repoName)
				kick(repoName)
			}
		}
	}()

	unfinishedMsg := "marked as failed/unfinished at ding startup."
	qStale := `
		with repo_builds as (
			select
				r.name as repoName,
				b.id as buildID
			from build b
			join repo r on b.repo_id = r.id
			where b.finish is null and b.status!='new'
		)
		select coalesce(json_agg(rb.*), '[]')
		from repo_builds rb
	`
	var stales []struct {
		RepoName string
		BuildID  int
	}
	checkRow(database.QueryRow(qStale), &stales, "looking for stale builds in database")
	for _, stale := range stales {
		buildDir := fmt.Sprintf("%s/build/%s/%d/", dingDataDir, stale.RepoName, stale.BuildID)
		du := buildDiskUsage(buildDir)

		qMarkStale := `update build set finish=now(), error_message=$1, disk_usage=$2 where finish is null and status!='new' returning id`
		checkRow(database.QueryRow(qMarkStale, unfinishedMsg, du), &stale.BuildID, "marking stale build in database")
		log.Printf("marked %s stale build as failed\n", buildDir)
	}

	var newBuilds []struct {
		Repo  Repo
		Build Build
	}
	qnew := `
		select coalesce(json_agg(x.*), '[]') from (
			select row_to_json(repo.*) as repo, row_to_json(build.*) as build from repo join build on repo.id = build.repo_id where status='new'
		) x
	`
	checkRow(database.QueryRow(qnew), &newBuilds, "fetching new builds from database")
	for _, repoBuild := range newBuilds {
		func(repo Repo, build Build) {
			job := job{
				repo.Name,
				make(chan struct{}),
			}
			newJobs <- job
			go func() {
				<-job.rc
				defer func() {
					finishedJobs <- job.repoName
				}()

				buildDir := fmt.Sprintf("%s/build/%s/%d", dingDataDir, repo.Name, build.ID)
				_doBuild(repo, build, buildDir)
			}()
		}(repoBuild.Repo, repoBuild.Build)
	}

	if *listenWebhookAddress != "" {
		log.Printf("ding version %s, listening on %s and for webhooks on %s\n", version, *listenAddress, *listenWebhookAddress)
		webhookHandler := http.NewServeMux()
		webhookHandler.HandleFunc("/github/", githubHookHandler)
		webhookHandler.HandleFunc("/bitbucket/", bitbucketHookHandler)
		go func() {
			server := &http.Server{Addr: *listenWebhookAddress, Handler: webhookHandler}
			log.Fatal(server.ListenAndServe())
		}()
	} else {
		log.Printf("ding version %s, listening on %s\n", version, *listenAddress)
	}
	go func() {
		log.Fatal(http.ListenAndServe(*listenAddress, nil))
	}()

	enc := gob.NewEncoder(msgfile)
	for {
		req := <-rootRequests
		err = enc.Encode(req.msg)
		check(err, "writing msg to root")

		var r string
		err = dec.Decode(&r)
		check(err, "reading response from root")

		switch {
		case req.msg.Build != nil:
			if r != "" {
				err = fmt.Errorf("%s", r)
				log.Println("run failed:", err)
				req.buildResponse <- buildResult{err, nil, nil, nil}
				continue
			}

			buf := make([]byte, 1)   // nothing in there
			oob := make([]byte, 128) // expect 3*24 bytes
			_, oobn, _, _, err := unixconn.ReadMsgUnix(buf, oob)
			check(err, "receiving fd")
			scms, err := unix.ParseSocketControlMessage(oob[:oobn])
			check(err, "parsing control message")
			if len(scms) != 1 {
				log.Fatalln("client: expected 1 SocketControlMessage; got scms =", scms)
			}

			fds, err := unix.ParseUnixRights(&scms[0])
			check(err, "parse unix rights")
			if len(fds) != 3 {
				log.Fatalf("wanted 3 fds; got %d fds\n", len(fds))
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

func serveAsset(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/") {
		r.URL.Path += "index.html"
	}
	f, err := httpFS.Open("/web" + r.URL.Path)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		log.Printf("serving asset %s: %s\n", r.URL.Path, err)
		http.Error(w, "500 - Server error", 500)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		log.Printf("serving asset %s: %s\n", r.URL.Path, err)
		http.Error(w, "500 - Server error", 500)
		return
	}

	if info.IsDir() {
		http.NotFound(w, r)
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
	if r.Method != "GET" {
		http.Error(w, "bad method", 405)
		return
	}
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
		http.Error(w, "server error", 500)
		return
	}
	defer f.Close()

	if acceptsGzip(r.Header.Get("Accept-Encoding")) {
		w.Header().Set("Content-Encoding", "gzip")
		io.Copy(w, f) // nothing to do for errors
	} else {
		gzr, err := gzip.NewReader(f)
		if err != nil {
			log.Printf("release: reading gzip file %s: %s\n", path, err)
			http.Error(w, "server error", 500)
			return
		}
		io.Copy(w, gzr) // nothing to do for errors
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
	if r.Method != "GET" {
		http.Error(w, "bad method", 405)
		return
	}
	t := strings.Split(r.URL.Path[1:], "/")
	if len(t) != 4 || hasBadElems(t[1:]) {
		http.NotFound(w, r)
		return
	}
	repoName := t[1]
	buildID, err := strconv.Atoi(t[2])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	filename := t[3]

	q := `
		select row_to_json(x.*)
		from (
			select
				repo.checkout_path,
				coalesce(json_agg(result.filename), '[]') as filenames
			from result
			join build on result.build_id = build.id
			join repo on build.repo_id = repo.id
			where repo.name=$1 and build.id=$2
			group by repo.checkout_path
		) x
	`
	var buildResults struct {
		CheckoutPath string   `json:"checkout_path"`
		Filenames    []string `json:"filenames"`
	}
	var buf []byte
	err = database.QueryRow(q, repoName, buildID).Scan(&buf)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err == nil {
		err = json.Unmarshal(buf, &buildResults)
	}
	if err != nil {
		log.Printf("fetching build results from database: %s", err)
		http.Error(w, "500 internal error", http.StatusInternalServerError)
		return
	}
	suffix := "/" + filename
	for _, path := range buildResults.Filenames {
		if path == filename || strings.HasSuffix(filename, suffix) {
			p := fmt.Sprintf("%s/build/%s/%d/checkout/%s/%s", dingDataDir, repoName, buildID, buildResults.CheckoutPath, path)
			http.ServeFile(w, r, p)
			return
		}
	}
	http.NotFound(w, r)
}
