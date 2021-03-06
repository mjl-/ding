package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mjl-/sherpa"
)

// We run commands under a context, so we can cancel is from the UI. The context
// for these commands are separate from the regular context that is used for
// requests to initiate builds. Both the root serve process as the ding http-serve
// process have their own bookkeeping of commands running for a build. The ding
// httpserve process can run commands to clone a repository. The root serve process
// runs the actual builds.
type buildCommand struct {
	ctx    context.Context
	cancel func()
}

var buildIDCommands = struct {
	sync.Mutex
	commands map[int32]buildCommand
}{
	commands: make(map[int32]buildCommand),
}

// buildIDCommandRegister makes a new context for a buildID.
// Once the build is done, buildIDCommandCancel must be called.
func buildIDCommandRegister(buildID int32) buildCommand {
	ctx, cancel := context.WithCancel(context.Background())
	bc := buildCommand{ctx, cancel}
	buildIDCommands.Lock()
	buildIDCommands.commands[buildID] = bc
	buildIDCommands.Unlock()
	return bc
}

// buildIDCommandCancel must be called to cleanup the context-with-cancel. It is
// also called to abort a running command.
func buildIDCommandCancel(buildID int32) {
	buildIDCommands.Lock()
	bc, ok := buildIDCommands.commands[buildID]
	delete(buildIDCommands.commands, buildID)
	buildIDCommands.Unlock()
	if ok {
		bc.cancel()
	}
}

func _prepareBuild(ctx context.Context, repoName, branch, commit string, lowPrio bool) (repo Repo, build Build, buildDir string) {
	transact(ctx, func(tx *sql.Tx) {
		repo = _repo(tx, repoName)

		q := `insert into build (repo_id, branch, commit_hash, status, low_prio) values ($1, $2, $3, $4, $5) returning id`
		sherpaCheckRow(tx.QueryRow(q, repo.ID, branch, commit, "new", lowPrio), &build.ID, "inserting new build into database")

		buildDir = fmt.Sprintf("%s/build/%s/%d", dingDataDir, repo.Name, build.ID)
		err := os.MkdirAll(buildDir, 0777)
		sherpaCheck(err, "creating build dir")

		homeDir := buildDir + "/home"
		if repo.UID != nil {
			homeDir = fmt.Sprintf("%s/home/%s", dingDataDir, repo.Name)
		}

		err = os.MkdirAll(buildDir+"/scripts", 0777)
		sherpaCheck(err, "creating scripts dir")
		err = os.MkdirAll(homeDir, 0777)
		sherpaCheck(err, "creating home dir")

		buildSh := buildDir + "/scripts/build.sh"
		writeFile(buildSh, repo.BuildScript)
		err = os.Chmod(buildSh, os.FileMode(0755))
		sherpaCheck(err, "chmod")

		outputDir := buildDir + "/output"
		err = os.MkdirAll(outputDir, 0777)
		sherpaCheck(err, "creating output dir")

		downloadDir := buildDir + "/dl"
		err = os.MkdirAll(downloadDir, 0777)
		sherpaCheck(err, "creating download dir")

		build = _build(tx, repo.Name, build.ID)
	})
	events <- EventBuild{repo.Name, build}
	return
}

func writeFile(path, content string) {
	f, err := os.Create(path)
	sherpaCheck(err, "creating file")
	_, err = f.Write([]byte(content))
	err2 := f.Close()
	if err == nil {
		err = err2
	}
	sherpaCheck(err, "writing file")
}

func prepareBuild(ctx context.Context, repoName, branch, commit string, lowPrio bool) (repo Repo, build Build, buildDir string, err error) {
	if branch == "" {
		err = fmt.Errorf("branch cannot be empty")
		return
	}
	defer func() {
		xerr := recover()
		if xerr == nil {
			return
		}
		if serr, ok := xerr.(*sherpa.Error); ok {
			err = fmt.Errorf("%s", serr.Error())
		} else {
			panic(xerr)
		}
	}()
	repo, build, buildDir = _prepareBuild(ctx, repoName, branch, commit, lowPrio)
	return repo, build, buildDir, nil
}

func doBuild(ctx context.Context, repo Repo, build Build, buildDir string) {
	job := job{
		repo.Name,
		build.LowPrio,
		make(chan struct{}),
	}
	newJobs <- job
	<-job.rc
	defer func() {
		finishedJobs <- job.repoName
	}()
	_doBuild(ctx, repo, build, buildDir)
}

func _doBuild(ctx context.Context, repo Repo, build Build, buildDir string) {
	buildCmd := buildIDCommandRegister(build.ID)
	defer buildIDCommandCancel(build.ID)

	// We may have been cancelled in the mean time. If so, do not even start working.
	var abort bool
	transact(ctx, func(tx *sql.Tx) {
		q := `select finish is not null from build where id=$1`
		sherpaCheckRow(tx.QueryRow(q, &build.ID), &abort, "fetching finish status from database")
	})
	if abort {
		return
	}

	var homeDir string
	if repo.UID != nil {
		homeDir = fmt.Sprintf("%s/home/%s", dingDataDir, repo.Name)
	} else {
		homeDir = fmt.Sprintf("%s/home", buildDir)
	}

	defer func() {
		build.DiskUsage = buildDiskUsage(buildDir)

		var homeDiskUsage, homeDiskUsageDelta int64
		if repo.UID != nil {
			homeDiskUsage = buildDiskUsage(homeDir)
			homeDiskUsageDelta = homeDiskUsage - repo.HomeDiskUsage
		}

		transact(ctx, func(tx *sql.Tx) {
			var one int
			qBuild := `update build set finish=NOW(), disk_usage=$1, home_disk_usage_delta=$2 where id=$3 returning 1`
			err := tx.QueryRow(qBuild, build.DiskUsage, homeDiskUsageDelta, build.ID).Scan(&one)
			sherpaCheck(err, "marking build as finished in database")

			if repo.UID != nil {
				qRepo := `update repo set home_disk_usage=$1 where id=$2 returning 1`
				err := tx.QueryRow(qRepo, homeDiskUsage, repo.ID).Scan(&one)
				sherpaCheck(err, "storing home directory disk usage in database")
			}

			events <- EventBuild{repo.Name, _build(tx, repo.Name, build.ID)}
		})

		_cleanupBuilds(ctx, repo.Name, build.Branch)

		r := recover()
		if r != nil {
			if serr, ok := r.(*sherpa.Error); ok && serr.Code == "userError" {
				transact(ctx, func(tx *sql.Tx) {
					err := tx.QueryRow(`update build set error_message=$1 where id=$2 returning id`, serr.Message, build.ID).Scan(&build.ID)
					sherpaCheck(err, "updating error message in database")
					events <- EventBuild{repo.Name, _build(tx, repo.Name, build.ID)}
				})
			} else {
				panic(r)
			}
		}

		var prevStatus string
		err := database.QueryRowContext(ctx, "select status from build join repo on build.repo_id = repo.id and repo.name = $1 and build.branch = $2 order by build.id desc offset 1 limit 1", repo.Name, build.Branch).Scan(&prevStatus)
		if r != nil && (err != nil || prevStatus == "success") {

			// for build.LastLine
			transact(ctx, func(tx *sql.Tx) {
				build = _build(tx, repo.Name, build.ID)
			})
			fillBuild(repo.Name, &build)

			var errmsg string
			if serr, ok := r.(*sherpa.Error); ok {
				errmsg = serr.Message
			} else {
				errmsg = fmt.Sprintf("%v", r)
			}
			_sendMailFailing(repo, build, errmsg)
		}
		if r == nil && err == nil && prevStatus != "success" {
			_sendMailFixed(repo, build)
		}

		if r != nil {
			if serr, ok := r.(*sherpa.Error); !ok || serr.Code != "userError" {
				panic(r)
			}
		}
	}()

	_updateStatus := func(status string, isStart bool) {
		transact(ctx, func(tx *sql.Tx) {
			var one int
			err := tx.QueryRow("update build set status=$1 where id=$2 returning 1", status, build.ID).Scan(&one)
			sherpaCheck(err, "updating build status in database")

			if isStart {
				q := "update build set start=now() where id=$1 returning 1"
				sherpaCheckRow(tx.QueryRow(q, build.ID), &one, "marking start time for build in database")
			}

			events <- EventBuild{repo.Name, _build(tx, repo.Name, build.ID)}
		})
	}

	env := []string{
		"HOME=" + homeDir,
		"DING_BUILDDIR=" + buildDir,
		"DING_CHECKOUTPATH=" + repo.CheckoutPath,
		"DING_DOWNLOADDIR=" + buildDir + "/dl",
		"DING_BUILDID=" + fmt.Sprintf("%d", build.ID),
		"DING_REPONAME=" + repo.Name,
		"DING_BRANCH=" + build.Branch,
		"DING_COMMIT=" + build.CommitHash,
	}
	env = append(env, config.Environment...)

	runPrefix := func(args ...string) []string {
		if len(config.Run) > 0 {
			args = append(config.Run, args...)
		}
		return args
	}

	_updateStatus("clone", true)
	var err error
	switch repo.VCS {
	case "git":
		// we clone without hard links because we chown later, don't want to mess up local git source repo's
		// we have to clone as the user running ding. otherwise, git clone won't work due to ssh refusing to run as a user without a username ("No user exists for uid ...")
		err = run(buildCmd.ctx, build.ID, env, "clone", buildDir, buildDir, runPrefix("git", "clone", "--recursive", "--no-hardlinks", "--branch", build.Branch, repo.Origin, "checkout/"+repo.CheckoutPath)...)
		sherpaUserCheck(err, "cloning git repository")
	case "mercurial":
		cmd := []string{"hg", "clone", "--branch", build.Branch}
		if build.CommitHash != "" {
			cmd = append(cmd, "--rev", build.CommitHash, "--updaterev", build.CommitHash)
		}
		cmd = append(cmd, repo.Origin, "checkout/"+repo.CheckoutPath)
		err = run(buildCmd.ctx, build.ID, env, "clone", buildDir, buildDir, runPrefix(cmd...)...)
		sherpaUserCheck(err, "cloning mercurial repository")
	case "command":
		err = run(buildCmd.ctx, build.ID, env, "clone", buildDir, buildDir, runPrefix("sh", "-c", repo.Origin)...)
		sherpaUserCheck(err, "cloning repository from command")
	default:
		serverError("unexpected VCS " + repo.VCS)
	}

	checkoutDir := fmt.Sprintf("%s/checkout/%s", buildDir, repo.CheckoutPath)

	if build.CommitHash == "" {
		if repo.VCS == "command" {
			clone := readFile(buildDir + "/output/clone.stdout")
			clone = strings.TrimSpace(clone)
			l := strings.Split(clone, "\n")
			s := l[len(l)-1]
			if !strings.HasPrefix(s, "commit:") {
				userError(`output of clone command should start with "commit:" followed by the commit id/hash`)
			}
			build.CommitHash = s[len("commit:"):]
		} else {
			var command []string
			switch repo.VCS {
			case "git":
				command = []string{"git", "rev-parse", "HEAD"}
			case "mercurial":
				command = []string{"hg", "id", "--id"}
			default:
				serverError("unexpected VCS " + repo.VCS)
			}

			argv := runPrefix(command...)
			cmd := exec.CommandContext(buildCmd.ctx, argv[0], argv[1:]...)
			cmd.Dir = checkoutDir
			buf, err := cmd.Output()
			sherpaCheck(err, "finding commit hash")
			build.CommitHash = strings.TrimSpace(string(buf))
		}
		if build.CommitHash == "" {
			sherpaCheck(fmt.Errorf("cannot find commit hash"), "finding commit hash")
		}
		transact(ctx, func(tx *sql.Tx) {
			err = tx.QueryRow(`update build set commit_hash=$1 where id=$2 returning id`, build.CommitHash, build.ID).Scan(&build.ID)
			sherpaCheck(err, "updating commit hash in database")
			events <- EventBuild{repo.Name, _build(tx, repo.Name, build.ID)}
		})
	}

	if repo.VCS == "git" {
		err = run(buildCmd.ctx, build.ID, env, "clone", buildDir, checkoutDir, runPrefix("git", "checkout", build.CommitHash)...)
		sherpaUserCheck(err, "checkout revision")
	}

	var uid uint32
	SharedHome := false
	if config.IsolateBuilds.Enabled {
		if repo.UID != nil {
			uid = *repo.UID
			SharedHome = true
		} else {
			uid = config.IsolateBuilds.UIDStart + uint32(build.ID)%(config.IsolateBuilds.UIDEnd-config.IsolateBuilds.UIDStart)
		}
	}

	chownMsg := msg{Chown: &msgChown{repo.Name, build.ID, SharedHome, uid}}
	err = requestPrivileged(chownMsg)
	sherpaCheck(err, "chown")

	_updateStatus("build", false)
	req := request{
		msg{Build: &msgBuild{repo.Name, build.ID, uid, repo.CheckoutPath, env}},
		nil,
		make(chan buildResult),
	}
	rootRequests <- req
	result := <-req.buildResponse
	if result.err != nil {
		sherpaUserCheck(result.err, "building")
	}

	wait := make(chan error, 1)
	go func() {
		defer result.status.Close()

		var r string
		err = gob.NewDecoder(result.status).Decode(&r)
		check(err, "decoding gob from result.status")
		var err error
		if r != "" {
			err = fmt.Errorf("%s", r)
		}
		wait <- err
	}()
	err = track(build.ID, "build", buildDir, result.stdout, result.stderr, wait)
	sherpaUserCheck(err, "build.sh")

	transact(ctx, func(tx *sql.Tx) {
		outputDir := buildDir + "/output"
		version, results, coverage, coverageReportFile := parseResults(repo, build, checkoutDir, outputDir+"/build.stdout")

		qins := `insert into result (build_id, command, os, arch, toolchain, filename, filesize) values ($1, $2, $3, $4, $5, $6, $7) returning id`
		for _, result := range results {
			var id int
			err = tx.QueryRow(qins, build.ID, result.Command, result.Os, result.Arch, result.Toolchain, result.Filename, result.Filesize).Scan(&id)
			sherpaCheck(err, "inserting result into database")
		}

		var one int
		err = tx.QueryRow("update build set status='success', coverage=$1, coverage_report_file=$2, version=$3 where id=$4 returning 1", coverage, coverageReportFile, version, build.ID).Scan(&one)
		sherpaCheck(err, "marking build as success in database")

		events <- EventBuild{repo.Name, _build(tx, repo.Name, build.ID)}
	})
}

func _cleanupBuilds(ctx context.Context, repoName, branch string) {
	var builds []Build
	q := `
		select coalesce(json_agg(x.* order by x.id desc), '[]')
		from (
			select build.*
			from build join repo on build.repo_id = repo.id
			where repo.name=$1 and build.branch=$2
		) x
	`
	sherpaCheckRow(database.QueryRowContext(ctx, q, repoName, branch), &builds, "fetching builds from database")
	now := time.Now()
	for index, b := range builds {
		if index == 0 || b.Released != nil {
			continue
		}
		if index >= 10 || (b.Finish != nil && now.Sub(*b.Finish) > 14*24*3600*time.Second) {
			transact(ctx, func(tx *sql.Tx) {
				_removeBuild(tx, repoName, b.ID)
			})
			events <- EventRemoveBuild{repoName, b.ID}
		}
	}
}

func parseResults(repo Repo, build Build, checkoutDir, path string) (version string, results []Result, coverage *float32, coverageReportFile string) {
	f, err := os.Open(path)
	sherpaUserCheck(err, "opening build output")
	defer func() {
		sherpaUserCheck(f.Close(), "closing build output")
	}()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		t := strings.Split(line, " ")
		switch t[0] {
		case "release:":
			//  "release:" command version os arch toolchain path
			if len(t) != 6 {
				userError("invalid \"release:\"-line, should have 6 words: " + line)
			}
			result := Result{t[1], t[2], t[3], t[4], t[5], 0}
			if !strings.HasPrefix(result.Filename, "/") {
				result.Filename = checkoutDir + "/" + result.Filename
			}
			info, err := os.Stat(result.Filename)
			sherpaUserCheck(err, "testing whether released file exists")
			result.Filename = result.Filename[len(checkoutDir+"/"):]
			result.Filesize = info.Size()
			results = append(results, result)
		case "version:":
			if len(t) != 2 {
				userError("invalid \"version:\"-line, should have 1 parameter: " + line)
			}
			version = t[1]
		case "coverage:":
			// "coverage:" 75.0
			if len(t) != 2 {
				userError("invalid \"coverage:\"-line, should have 1 parameter: " + line)
			}
			var fl float64
			fl, err = strconv.ParseFloat(t[1], 32)
			if err != nil {
				userError(fmt.Sprintf("invalid \"coverage:\"-line (%q), parsing float: %s", line, err))
			}
			coverage = new(float32)
			*coverage = float32(fl)
		case "coverage-report:":
			// "coverage-report:" coverage.html
			if len(t) != 2 {
				userError("invalid \"coverage-report:\"-line, should have 1 parameter: " + line)
			}
			coverageReportFile = t[1]
			p := fmt.Sprintf("%s/build/%s/%d/dl/%s", dingDataDir, repo.Name, build.ID, coverageReportFile)
			_, err := os.Stat(p)
			if err != nil {
				userError(fmt.Sprintf("bad file in \"coverage-report:\"-line (%q): %s", line, err))
			}
		}
	}
	err = scanner.Err()
	sherpaUserCheck(err, "reading build output")
	return
}

// start a command and return readers for its output and the final result of the command.
// it mimics a command started through the root process under a unique uid.
func setupCmd(cmdCtx context.Context, buildID int32, env []string, step, buildDir, workDir string, args ...string) (stdout, stderr io.ReadCloser, wait <-chan error, rerr error) {
	type Error struct {
		err error
	}

	var stdoutr, stdoutw, stderrr, stderrw *os.File
	defer func() {
		close := func(f *os.File) {
			if f != nil {
				f.Close()
			}
		}
		// always close subprocess-part of the fd's
		close(stdoutw)
		close(stderrw)

		e := recover()
		if e == nil {
			return
		}

		if ee, ok := e.(Error); ok {
			// only close returning fd's on error
			close(stdoutr)
			close(stderrr)

			rerr = ee.err
			return
		}
		panic(e)
	}()

	xcheck := func(err error, msg string) {
		if err != nil {
			panic(Error{fmt.Errorf("%s: %s", msg, err)})
		}
	}

	var err error
	stdoutr, stdoutw, err = os.Pipe()
	xcheck(err, "pipe for stdout")

	stderrr, stderrw, err = os.Pipe()
	xcheck(err, "pipe for stderr")

	cmd := exec.CommandContext(cmdCtx, args[0], args...)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Stdout = stdoutw
	cmd.Stderr = stderrw

	err = cmd.Start()
	xcheck(err, "starting command")

	c := make(chan error, 1)
	go func() {
		c <- cmd.Wait()
	}()
	return stdoutr, stderrr, c, nil
}

func run(cmdCtx context.Context, buildID int32, env []string, step, buildDir, workDir string, args ...string) error {
	cmdstdout, cmdstderr, wait, err := setupCmd(cmdCtx, buildID, env, step, buildDir, workDir, args...)
	if err != nil {
		return fmt.Errorf("setting up command: %s", err)
	}
	return track(buildID, step, buildDir, cmdstdout, cmdstderr, wait)
}

func track(buildID int32, step, buildDir string, cmdstdout, cmdstderr io.ReadCloser, wait <-chan error) (rerr error) {
	type Error struct {
		err error
	}

	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if ee, ok := e.(Error); ok {
			rerr = ee.err
			return
		}
		panic(e)
	}()

	xcheck := func(err error, msg string) {
		if err != nil {
			panic(Error{fmt.Errorf("%s: %s", msg, err)})
		}
	}

	defer func() {
		cmdstdout.Close()
		cmdstderr.Close()
	}()

	// write .nsec file when we're done here
	t0 := time.Now()
	defer func() {
		time.Since(t0)
		nsec, err := os.Create(buildDir + "/output/" + step + ".nsec")
		xcheck(err, "creating nsec file")
		defer nsec.Close()
		_, err = fmt.Fprintf(nsec, "%d", time.Since(t0))
		xcheck(err, "writing nsec file")
	}()

	appendFlags := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	output, err := os.OpenFile(buildDir+"/output/"+step+".output", appendFlags, 0644)
	xcheck(err, "creating output file")
	defer output.Close()
	stdout, err := os.OpenFile(buildDir+"/output/"+step+".stdout", appendFlags, 0644)
	xcheck(err, "creating stdout file")
	defer stdout.Close()
	stderr, err := os.OpenFile(buildDir+"/output/"+step+".stderr", appendFlags, 0644)
	xcheck(err, "creating stderr file")
	defer stderr.Close()

	// let it be known that we started this phase
	events <- EventOutput{buildID, step, "stdout", ""}

	// first we read all the data from stdout & stderr
	type Lines struct {
		text   string
		stdout bool
		err    error
	}
	lines := make(chan Lines)
	linereader := func(r io.ReadCloser, stdout bool) {
		buf := make([]byte, 1024)
		have := 0
		for {
			//log.Println("calling read")
			n, err := r.Read(buf[have:])
			//log.Println("read returned")
			if n > 0 {
				have += n
				end := bytes.LastIndexByte(buf[:have], '\n')
				if end < 0 && have == len(buf) {
					// cannot gather any more data, flush it
					end = len(buf)
				} else if end < 0 {
					continue
				} else {
					// include the newline
					end++
				}
				lines <- Lines{string(buf[:end]), stdout, nil}
				copy(buf[:], buf[end:have])
				have -= end
			}
			if err == io.EOF {
				lines <- Lines{"", stdout, nil}
				break
			}
			if err != nil {
				lines <- Lines{stdout: stdout, err: err}
				return
			}
		}
	}
	//log.Println("new command, reading input")
	go linereader(cmdstdout, true)
	go linereader(cmdstderr, false)
	eofs := 0
	for {
		l := <-lines
		//log.Println("have line", l)
		if l.text == "" || l.err != nil {
			if l.err != nil {
				log.Println("reading output from command:", l.err)
			}
			eofs++
			if eofs >= 2 {
				//log.Println("done with command output")
				break
			}
			continue
		}
		_, err = output.Write([]byte(l.text))
		xcheck(err, "writing to output")
		var where string
		if l.stdout {
			where = "stdout"
			_, err = stdout.Write([]byte(l.text))
			xcheck(err, "writing to stdout")
		} else {
			where = "stderr"
			_, err = stderr.Write([]byte(l.text))
			xcheck(err, "writing to stderr")
		}
		events <- EventOutput{buildID, step, where, l.text}
	}

	// second, we wait for the command result
	return <-wait
}

// disk usage, best effort
func buildDiskUsage(buildDir string) (diskUsage int64) {
	filepath.Walk(buildDir, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			const overhead = 2 * 1024
			diskUsage += overhead + info.Size()
		}
		return nil
	})
	return
}
