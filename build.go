package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mjl-/bstore"
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
		slog.Debug("canceling build command", "buildid", buildID)
		bc.cancel()
	}
}

func _prepareBuild(ctx context.Context, repoName, branch, commit string, lowPrio bool) (repo Repo, build Build, buildDir string) {
	_dbwrite(ctx, func(tx *bstore.Tx) {
		repo = _repo(tx, repoName)

		b := Build{
			RepoName:    repo.Name,
			Branch:      branch,
			CommitHash:  commit,
			Status:      StatusNew,
			LowPrio:     lowPrio,
			BuildScript: repo.BuildScript,
		}
		err := tx.Insert(&b)
		_checkf(err, "inserting new build into database")

		buildDir = fmt.Sprintf("%s/build/%s/%d", dingDataDir, repo.Name, b.ID)
		err = os.MkdirAll(buildDir, 0777)
		_checkf(err, "creating build dir")

		homeDir := buildDir + "/home"
		if repo.UID != nil {
			homeDir = fmt.Sprintf("%s/home/%s", dingDataDir, repo.Name)
		}

		err = os.MkdirAll(buildDir+"/scripts", 0777)
		_checkf(err, "creating scripts dir")
		err = os.MkdirAll(homeDir, 0777)
		_checkf(err, "creating home dir")

		buildSh := buildDir + "/scripts/build.sh"
		_writeFile(buildSh, repo.BuildScript)
		err = os.Chmod(buildSh, os.FileMode(0755))
		_checkf(err, "chmod")

		outputDir := buildDir + "/output"
		err = os.MkdirAll(outputDir, 0777)
		_checkf(err, "creating output dir")

		downloadDir := buildDir + "/dl"
		err = os.MkdirAll(downloadDir, 0777)
		_checkf(err, "creating download dir")

		build = b
	})
	events <- EventBuild{build}
	return
}

func _writeFile(path, content string) {
	f, err := os.Create(path)
	_checkf(err, "creating file")
	_, err = f.Write([]byte(content))
	err2 := f.Close()
	if err == nil {
		err = err2
	}
	_checkf(err, "writing file")
}

func prepareBuild(ctx context.Context, repoName, branch, commit string, lowPrio bool) (repo Repo, build Build, buildDir string, err error) {
	if branch == "" {
		err = fmt.Errorf("branch cannot be empty")
		return
	}
	defer func() {
		if x := recover(); x != nil {
			if xerr, ok := x.(error); ok {
				err = xerr
			} else {
				err = fmt.Errorf("%v", x)
			}
		}
	}()
	repo, build, buildDir = _prepareBuild(ctx, repoName, branch, commit, lowPrio)
	return repo, build, buildDir, nil
}

func doBuild(ctx context.Context, repo Repo, build Build, buildDir string) (rerr error) {
	defer func() {
		if x := recover(); x != nil {
			if err, ok := x.(*sherpa.Error); ok {
				rerr = fmt.Errorf("%s (%s)", err.Message, err.Code)
			} else {
				rerr = fmt.Errorf("%v", x)
			}
		}
	}()
	_doBuild(ctx, repo, build, buildDir)
	return nil
}

func _doBuild(ctx context.Context, repo Repo, build Build, buildDir string) {
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
	_doBuild0(ctx, repo, build, buildDir)
}

func _doBuild0(ctx context.Context, repo Repo, build Build, buildDir string) {
	slog.Debug("building", "repo", repo.Name, "buildid", build.ID)

	buildCmd := buildIDCommandRegister(build.ID)
	defer buildIDCommandCancel(build.ID)

	// We may have been cancelled in the mean time. If so, do not even start working.
	b := Build{ID: build.ID}
	err := database.Get(ctx, &b)
	_checkf(err, "fetching build from database for finish status")
	if b.Finish != nil {
		return
	}

	settings := Settings{ID: 1}
	err = database.Get(ctx, &settings)
	_checkf(err, "get settings")

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

		var b Build
		_dbwrite(ctx, func(tx *bstore.Tx) {
			b = Build{ID: build.ID}
			err := tx.Get(&b)
			_checkf(err, "get build after finish")
			setLastLine(&b)
			now := time.Now()
			b.Finish = &now
			b.DiskUsage = build.DiskUsage
			b.HomeDiskUsageDelta = homeDiskUsageDelta
			b.Steps = _buildSteps(b)
			err = tx.Update(&b)
			_checkf(err, "marking build as finished in database")

			if repo.UID != nil {
				r := Repo{Name: repo.Name}
				err := tx.Get(&r)
				_checkf(err, "get repo")
				r.HomeDiskUsage = homeDiskUsage
				err = tx.Update(&r)
				_checkf(err, "storing home directory disk usage in database")
			}

		})
		events <- EventBuild{b}

		_cleanupBuilds(ctx, repo.Name, build.Branch)

		r := recover()
		if r != nil {
			if serr, ok := r.(*sherpa.Error); ok && serr.Code == "user:error" {
				_dbwrite(ctx, func(tx *bstore.Tx) {
					b = Build{ID: build.ID}
					err := tx.Get(&b)
					_checkf(err, "get build after error")
					b.ErrorMessage = serr.Message
					err = tx.Update(&b)
					_checkf(err, "update error message for build in database")
				})
				events <- EventBuild{b}
			} else {
				panic(r)
			}
		}

		// Get previous build status for same repo/branch, and send email when this breaks
		// or fixes the build for this branch.
		var prevStatus BuildStatus
		_dbread(ctx, func(tx *bstore.Tx) {
			q := bstore.QueryTx[Build](tx).FilterNonzero(Build{Branch: build.Branch, RepoName: repo.Name}).SortDesc("ID")
			_, err := q.Next()
			if err == bstore.ErrAbsent {
				return
			}
			_checkf(err, "get build for branch")
			b, err := q.Next()
			if err == bstore.ErrAbsent {
				return
			}
			prevStatus = b.Status
		})
		if r != nil && (prevStatus == "" || prevStatus == StatusSuccess) {
			var errmsg string
			if serr, ok := r.(*sherpa.Error); ok {
				errmsg = serr.Message
			} else {
				errmsg = fmt.Sprintf("%v", r)
			}
			_sendMailFailing(settings, repo, build, errmsg)
		}
		if r == nil && !(prevStatus == "" || prevStatus == StatusSuccess) {
			_sendMailFixed(settings, repo, build)
		}

		if r != nil {
			if serr, ok := r.(*sherpa.Error); !ok || serr.Code != "user:error" {
				panic(r)
			}
		}
	}()

	_updateStatus := func(status BuildStatus, isStart bool) {
		_dbwrite(ctx, func(tx *bstore.Tx) {
			b := Build{ID: build.ID}
			err := tx.Get(&b)
			_checkf(err, "get build for status update")
			b.Status = status
			slog.Debug("updating build status", "buildid", build.ID, "status", status)

			if isStart {
				now := time.Now()
				b.Start = &now
			}

			err = tx.Update(&b)
			_checkf(err, "updating build status in database")

			events <- EventBuild{b}
		})
	}

	// Also see cmdbuild.go.
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
	var toolchainDir string
	if config.GoToolchainDir != "" {
		toolchainDir = config.GoToolchainDir
		if !path.IsAbs(toolchainDir) {
			workDir, err := os.Getwd()
			if err != nil {
				slog.Error("get workdir for toolchain dir", "err", err)
				toolchainDir = ""
			} else {
				toolchainDir = path.Join(workDir, toolchainDir)
			}
		}
		if toolchainDir != "" {
			env = append(env, "DING_TOOLCHAINDIR="+toolchainDir)
		}
	}
	env = append(env, settings.Environment...)

	runPrefix := func(args ...string) []string {
		if len(settings.RunPrefix) > 0 {
			args = append(settings.RunPrefix, args...)
		}
		return args
	}

	_updateStatus(StatusClone, true)
	switch repo.VCS {
	case VCSGit:
		// We clone without hard links because we chown later, don't want to mess up local
		// git source repo's. We have to clone as the user running ding. Otherwise, git
		// clone won't work due to ssh refusing to run as a user without a username ("No
		// user exists for uid ...")
		err = run(buildCmd.ctx, build.ID, settings.RunPrefix, env, "clone", buildDir, buildDir, runPrefix("git", "clone", "--recursive", "--no-hardlinks", "--branch", build.Branch, repo.Origin, "checkout/"+repo.CheckoutPath)...)
		_checkUserf(err, "cloning git repository")
	case VCSMercurial:
		cmd := []string{"hg", "clone", "--branch", build.Branch}
		if build.CommitHash != "" {
			cmd = append(cmd, "--rev", build.CommitHash, "--updaterev", build.CommitHash)
		}
		cmd = append(cmd, repo.Origin, "checkout/"+repo.CheckoutPath)
		err = run(buildCmd.ctx, build.ID, settings.RunPrefix, env, "clone", buildDir, buildDir, runPrefix(cmd...)...)
		_checkUserf(err, "cloning mercurial repository")
	case VCSCommand:
		err = run(buildCmd.ctx, build.ID, settings.RunPrefix, env, "clone", buildDir, buildDir, runPrefix("sh", "-c", repo.Origin)...)
		_checkUserf(err, "cloning repository from command")
	default:
		_serverError("unexpected VCS " + string(repo.VCS))
	}

	checkoutDir := fmt.Sprintf("%s/checkout/%s", buildDir, repo.CheckoutPath)

	if build.CommitHash == "" {
		if repo.VCS == VCSCommand {
			clone := _readFile(buildDir + "/output/clone.stdout")
			clone = strings.TrimSpace(clone)
			l := strings.Split(clone, "\n")
			s := l[len(l)-1]
			if !strings.HasPrefix(s, "commit:") {
				_userError(`output of clone command should start with "commit:" followed by the commit id/hash`)
			}
			build.CommitHash = s[len("commit:"):]
		} else {
			var command []string
			switch repo.VCS {
			case VCSGit:
				command = []string{"git", "rev-parse", "HEAD"}
			case VCSMercurial:
				command = []string{"hg", "id", "--id"}
			default:
				_serverError("unexpected VCS " + string(repo.VCS))
			}

			argv := runPrefix(command...)
			cmd := exec.CommandContext(buildCmd.ctx, argv[0], argv[1:]...)
			cmd.Dir = checkoutDir
			buf, err := cmd.Output()
			_checkf(err, "finding commit hash")
			build.CommitHash = strings.TrimSpace(string(buf))
		}
		if build.CommitHash == "" {
			_checkf(fmt.Errorf("cannot find commit hash"), "finding commit hash")
		}
		_dbwrite(ctx, func(tx *bstore.Tx) {
			b := Build{ID: build.ID}
			err := tx.Get(&b)
			_checkf(err, "get build to update commithash")
			b.CommitHash = build.CommitHash
			err = tx.Update(&b)
			_checkf(err, "update commit hash for build in database")
			events <- EventBuild{b}
		})
	}

	if repo.VCS == VCSGit {
		err = run(buildCmd.ctx, build.ID, settings.RunPrefix, env, "clone", buildDir, checkoutDir, runPrefix("git", "checkout", "--detach", build.CommitHash)...)
		_checkUserf(err, "checkout revision")
	}

	var uid uint32
	sharedHome := false
	if config.IsolateBuilds.Enabled {
		if repo.UID != nil {
			uid = *repo.UID
			sharedHome = true
		} else {
			uid = config.IsolateBuilds.UIDStart + uint32(build.ID)%(config.IsolateBuilds.UIDEnd-config.IsolateBuilds.UIDStart)
		}
	}

	chownMsg := msg{Chown: &msgChown{repo.Name, build.ID, sharedHome, uid}}
	err = requestPrivileged(chownMsg)
	_checkf(err, "chown")

	_updateStatus(StatusBuild, false)
	req := request{
		msg{Build: &msgBuild{repo.Name, build.ID, uid, repo.CheckoutPath, settings.RunPrefix, env, toolchainDir, homeDir, repo.Bubblewrap, repo.BubblewrapNoNet}},
		nil,
		make(chan buildResult),
	}
	rootRequests <- req
	result := <-req.buildResponse
	if result.err != nil {
		_checkUserf(result.err, "building")
	}

	wait := make(chan error, 1)
	go func() {
		defer result.status.Close()

		var r string
		err = gob.NewDecoder(result.status).Decode(&r)
		xcheckf(err, "decoding gob from result.status")
		var err error
		if r != "" {
			err = fmt.Errorf("%s", r)
		}
		wait <- err
	}()
	err = track(build.ID, "build", buildDir, result.stdout, result.stderr, wait)
	_checkUserf(err, "build.sh")

	outputFile, err := os.Open(buildDir + "/output/build.stdout")
	_checkUserf(err, "opening build output")
	defer func() {
		_checkUserf(outputFile.Close(), "closing build output")
	}()

	dldir := path.Clean(fmt.Sprintf("%s/build/%s/%d/dl", dingDataDir, repo.Name, build.ID))
	version, results, coverage, coverageReportFile, err := parseResults(checkoutDir, dldir, outputFile)
	_checkUserf(err, "parse results from output")

	_dbwrite(ctx, func(tx *bstore.Tx) {
		b = Build{ID: build.ID}
		err := tx.Get(&b)
		_checkf(err, "get build to add results")
		b.Status = StatusSuccess
		b.Coverage = coverage
		b.CoverageReportFile = coverageReportFile
		b.Version = version
		b.Results = results
		err = tx.Update(&b)
		_checkf(err, "marking build as success in database")
		slog.Debug("updating build status", "buildid", build.ID, "status", b.Status)
	})
	events <- EventBuild{b}
}

func _cleanupBuilds(ctx context.Context, repoName, branch string) {
	var repo Repo
	var builds []Build
	_dbread(ctx, func(tx *bstore.Tx) {
		repo = _repo(tx, repoName)

		var err error
		builds, err = bstore.QueryTx[Build](tx).FilterNonzero(Build{RepoName: repo.Name}).SortDesc("ID").List()
		_checkf(err, "listing builds")
	})
	branchBuilds := map[string]int{} // Number of builds for branch.
	for _, b := range builds {
		if b.Finish == nil {
			continue
		}
		// For release builds, we cleanup the builddir after 60 days.
		if b.Released != nil {
			if !b.BuilddirRemoved && time.Since(*b.Finish) > 60*24*time.Hour {
				_dbwrite(ctx, func(tx *bstore.Tx) {
					_removeBuildDir(b)
					_, bb := _build(tx, b.RepoName, b.ID)
					bb.BuilddirRemoved = true
					err := tx.Update(&bb)
					_checkf(err, "marking build directory as removed")
				})
			}
			// For other builds, we keep max 10 of the latest builds per branch, but only if
			// not older than 30 days, although we keep at least 1 for the default branch of
			// the repo.
		} else if branchBuilds[b.Branch] >= 10 || time.Since(*b.Finish) > 30*24*time.Hour && (repo.DefaultBranch != b.Branch || branchBuilds[b.Branch] > 0) {
			_dbwrite(ctx, func(tx *bstore.Tx) {
				_removeBuild(tx, repoName, b.ID)
			})
			events <- EventRemoveBuild{repoName, b.ID}
			continue
		}
		branchBuilds[b.Branch]++
	}
}

func parseResults(checkoutDir, dldir string, r io.Reader) (version string, results []Result, coverage *float32, coverageReportFile string, rerr error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		t := strings.Split(line, " ")
		switch t[0] {
		case "release:":
			//  "release:" command version os arch toolchain path
			if len(t) != 6 {
				rerr = errors.New("invalid \"release:\"-line, should have 6 words: " + line)
				return
			}
			result := Result{t[1], t[2], t[3], t[4], path.Clean(t[5]), 0}
			if !path.IsAbs(result.Filename) {
				result.Filename = path.Join(checkoutDir, result.Filename)
			}
			if !strings.HasPrefix(result.Filename, path.Clean(checkoutDir)+"/") {
				rerr = errors.New("result file must be in checkout directory")
				return
			}
			info, err := os.Stat(result.Filename)
			if err != nil {
				rerr = errors.New("testing whether released file exists")
				return
			}
			result.Filename = result.Filename[len(checkoutDir+"/"):]
			result.Filesize = info.Size()
			results = append(results, result)
		case "version:":
			if len(t) != 2 {
				rerr = errors.New("invalid \"version:\"-line, should have 1 parameter: " + line)
				return
			}
			version = t[1]
		case "coverage:":
			// "coverage:" 75.0
			if len(t) != 2 {
				rerr = errors.New("invalid \"coverage:\"-line, should have 1 parameter: " + line)
				return
			}
			fl, err := strconv.ParseFloat(t[1], 32)
			if err != nil {
				rerr = fmt.Errorf("invalid \"coverage:\"-line (%q), parsing float: %s", line, err)
				return
			}
			coverage = new(float32)
			*coverage = float32(fl)
		case "coverage-report:":
			// "coverage-report:" coverage.html
			if len(t) != 2 {
				rerr = errors.New("invalid \"coverage-report:\"-line, should have 1 parameter: " + line)
				return
			}
			p := path.Clean(t[1])
			if !path.IsAbs(p) {
				p = path.Join(dldir, p)
			}
			if !strings.HasPrefix(p, dldir+"/") {
				rerr = errors.New("coverage file must be within $DING_DOWNLOADDIR")
				return
			}
			_, err := os.Stat(p)
			if err != nil {
				rerr = fmt.Errorf("bad file in \"coverage-report:\"-line (%q): %s", line, err)
				return
			}
		}
	}
	rerr = scanner.Err()
	return
}

// Start a command and return readers for its output and the final result of the command.
// It mimics a command started through the root process under a unique uid.
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
		// Always close subprocess-part of the fd's.
		close(stdoutw)
		close(stderrw)

		e := recover()
		if e == nil {
			return
		}

		if ee, ok := e.(Error); ok {
			// Only close returning fd's on error.
			close(stdoutr)
			close(stderrr)

			rerr = ee.err
			return
		}
		panic(e)
	}()

	lcheck := func(err error, msg string) {
		if err != nil {
			panic(Error{fmt.Errorf("%s: %s", msg, err)})
		}
	}

	var err error
	stdoutr, stdoutw, err = os.Pipe()
	lcheck(err, "pipe for stdout")

	stderrr, stderrw, err = os.Pipe()
	lcheck(err, "pipe for stderr")

	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.Stdout = stdoutw
	cmd.Stderr = stderrw

	err = cmd.Start()
	lcheck(err, "starting command")

	c := make(chan error, 1)
	go func() {
		c <- cmd.Wait()
	}()
	return stdoutr, stderrr, c, nil
}

func run(cmdCtx context.Context, buildID int32, runPrefix []string, env []string, step, buildDir, workDir string, args ...string) error {
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

	lcheck := func(err error, msg string) {
		if err != nil {
			panic(Error{fmt.Errorf("%s: %s", msg, err)})
		}
	}

	defer func() {
		cmdstdout.Close()
		cmdstderr.Close()
	}()

	// Write .nsec file when we're done here.
	t0 := time.Now()
	defer func() {
		time.Since(t0)
		nsec, err := os.Create(buildDir + "/output/" + step + ".nsec")
		lcheck(err, "creating nsec file")
		defer nsec.Close()
		_, err = fmt.Fprintf(nsec, "%d", time.Since(t0))
		lcheck(err, "writing nsec file")
	}()

	appendFlags := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	output, err := os.OpenFile(buildDir+"/output/"+step+".output", appendFlags, 0644)
	lcheck(err, "creating output file")
	defer output.Close()
	stdout, err := os.OpenFile(buildDir+"/output/"+step+".stdout", appendFlags, 0644)
	lcheck(err, "creating stdout file")
	defer stdout.Close()
	stderr, err := os.OpenFile(buildDir+"/output/"+step+".stderr", appendFlags, 0644)
	lcheck(err, "creating stderr file")
	defer stderr.Close()

	// Let it be known that we started this phase.
	events <- EventOutput{buildID, step, "stdout", ""}

	// First we read all the data from stdout & stderr.
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
			n, err := r.Read(buf[have:])
			if n > 0 {
				have += n
				end := bytes.LastIndexByte(buf[:have], '\n')
				if end < 0 && have == len(buf) {
					// Cannot gather any more data, flush it.
					end = len(buf)
				} else if end < 0 {
					continue
				} else {
					// Include the newline.
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
	go linereader(cmdstdout, true)
	go linereader(cmdstderr, false)
	eofs := 0
	for {
		l := <-lines
		if l.text == "" || l.err != nil {
			if l.err != nil {
				slog.Error("reading output from command", "err", l.err)
			}
			eofs++
			if eofs >= 2 {
				break
			}
			continue
		}
		_, err = output.Write([]byte(l.text))
		lcheck(err, "writing to output")
		var where string
		if l.stdout {
			where = "stdout"
			_, err = stdout.Write([]byte(l.text))
			lcheck(err, "writing to stdout")
		} else {
			where = "stderr"
			_, err = stderr.Write([]byte(l.text))
			lcheck(err, "writing to stderr")
		}
		events <- EventOutput{buildID, step, where, l.text}
	}

	// Second, we wait for the command result.
	return <-wait
}

// Disk usage, best effort.
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

// _buildSteps reads steps from disk, for storing in build after finish.
func _buildSteps(b Build) (steps []Step) {
	steps = []Step{}

	buildDir := fmt.Sprintf("%s/build/%s/%d/", dingDataDir, b.RepoName, b.ID)
	outputDir := buildDir + "output/"
	diskSteps := []BuildStatus{StatusClone, StatusBuild}
	for _, stepName := range diskSteps {
		base := outputDir + string(stepName)
		steps = append(steps, Step{
			Name:   string(stepName),
			Output: readFileLax(base + ".output"),
			Nsec:   parseInt(readFileLax(base + ".nsec")),
		})
		if stepName == b.Status {
			break
		}
	}
	return
}

func parseInt(s string) int64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	_checkf(err, "parsing integer")
	return v
}
