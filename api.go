package main

import (
	"compress/gzip"
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"io"
	"log/slog"
	"maps"
	mathrand "math/rand/v2"
	"os"
	"os/exec"
	"path"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mjl-/bstore"
	"github.com/mjl-/goreleases"
	"github.com/mjl-/sherpa"
)

// The Ding API lets you compile git branches, build binaries, run tests, and
// publish binaries.
type Ding struct {
	SSE SSE `sherpa:"Server-Sent Events"`
}

// Status checks the health of the application.
func (Ding) Status(ctx context.Context) {
	f, err := os.Create(dingDataDir + "/test")
	if err != nil {
		msg := fmt.Sprintf("creating file: %v", err)
		slog.Error("status", "err", msg)
		panic(&sherpa.InternalServerError{Code: "server:error", Message: msg})
	}
	f.Close()
	os.Remove(dingDataDir + "/test")
}

func _repo(tx *bstore.Tx, repoName string) Repo {
	r := Repo{Name: repoName}
	err := tx.Get(&r)
	_checkf(err, "get repo")
	return r
}

func _build(tx *bstore.Tx, repoName string, id int32) (Repo, Build) {
	r := Repo{Name: repoName}
	err := tx.Get(&r)
	_checkf(err, "get repo")
	b, err := bstore.QueryTx[Build](tx).FilterNonzero(Build{ID: id, RepoName: repoName}).Get()
	_checkf(err, "get build by id")
	return r, b
}

func _checkPassword(password string) {
	if password != config.Password {
		panic(&sherpa.Error{Code: "user:badAuth", Message: "bad password"})
	}
}

// BuildCreate builds a specific commit in the background, returning immediately.
//
// `Commit` can be empty, in which case the origin is cloned and the checked
// out commit is looked up.
//
// Low priority builds are executed after regular builds. And only one low
// priority build is running over all repo's.
func (Ding) BuildCreate(ctx context.Context, password, repoName, branch, commit string, lowPrio bool) Build {
	_checkPassword(password)

	if branch == "" {
		_userError("Branch cannot be empty")
	}

	repo, build, buildDir := _prepareBuild(ctx, repoName, branch, commit, lowPrio)
	go func() {
		defer func() {
			if x := recover(); x != nil {
				slog.Error("build", "err", x)
			}
		}()
		_doBuild(context.Background(), repo, build, buildDir)
	}()
	return build
}

// CreateBuild exists for compatibility with older "ding kick" behaviour.
func (Ding) CreateBuild(ctx context.Context, password, repoName, branch, commit string) Build {
	return Ding{}.BuildCreate(ctx, password, repoName, branch, commit, false)
}

// BuildsCreateLowPrio creates low priority builds for each repository, for the default branch.
func (Ding) BuildsCreateLowPrio(ctx context.Context, password string) {
	_checkPassword(password)

	err := scheduleLowPrioBuilds(ctx, false)
	_checkf(err, "scheduling low prio builds")
}

func scheduleLowPrioBuilds(ctx context.Context, automaticOnly bool) error {
	q := bstore.QueryDB[Repo](ctx, database)
	if automaticOnly {
		q = q.FilterNonzero(Repo{BuildOnUpdatedToolchain: true})
	}
	repos, err := q.List()
	if err != nil {
		return fmt.Errorf("fetching repo names from database: %v", err)
	}

	lowPrio := true
	commit := ""

	builds := make([]Build, len(repos))
	buildDirs := make([]string, len(repos))
	for i, repo := range repos {
		_, build, buildDir, err := prepareBuild(ctx, repo.Name, repo.DefaultBranch, commit, lowPrio)
		if err != nil {
			return fmt.Errorf("preparing build: %v", err)
		}
		builds[i] = build
		buildDirs[i] = buildDir
	}

	for i, repo := range repos {
		build := builds[i]
		buildDir := buildDirs[i]
		go func() {
			defer func() {
				if x := recover(); x != nil {
					slog.Error("lowprio build", "err", x)
				}
			}()
			_doBuild(context.Background(), repo, build, buildDir)
		}()
	}
	return nil
}

// BuildCancel cancels a currently running build.
func (Ding) BuildCancel(ctx context.Context, password, repoName string, buildID int32) {
	_checkPassword(password)

	_dbwrite(ctx, func(tx *bstore.Tx) {
		_, b := _build(tx, repoName, buildID)
		if b.Finish != nil {
			_userError("Build has already finished")
		}

		now := time.Now()
		b.Finish = &now
		b.Status = StatusCancelled
		b.Steps = _buildSteps(b)
		err := tx.Update(&b)
		_checkf(err, "marking build as cancelled in database")
	})

	// Cancel any commands in the http-serve process, like cloning.
	buildIDCommandCancel(buildID)

	// And cancel the actual build command controlled by the serve process.
	cancelMsg := msg{CancelCommand: &msgCancelCommand{buildID}}
	go func() {
		err := requestPrivileged(cancelMsg)
		if err != nil {
			slog.Error("requesting build cancel", "err", err)
		}
	}()
}

// BuildSettings describes the environment a build script is run in.
type BuildSettings struct {
	Run         []string // The command to run the build script is prefixed with these commands, e.g. /usr/bin/nice.
	Environment []string // Additional environment variables available during builds, of the form key=value.
}

// ReleaseCreate release a build.
func (Ding) ReleaseCreate(ctx context.Context, password, repoName string, buildID int32) (release Build) {
	_checkPassword(password)

	_dbwrite(ctx, func(tx *bstore.Tx) {
		r, b := _build(tx, repoName, buildID)
		if b.Finish == nil {
			_userError("Build has not finished yet")
		}
		if b.Status != StatusSuccess {
			_userError("Build was not successful")
		}
		if b.Released != nil {
			_userError("Build already released")
		}

		now := time.Now()
		b.Released = &now
		err := tx.Update(&b)
		_checkf(err, "marking build as released")

		checkoutDir := fmt.Sprintf("%s/build/%s/%d/checkout/%s", dingDataDir, r.Name, b.ID, r.CheckoutPath)
		for _, res := range b.Results {
			_fileCopy(checkoutDir+"/"+res.Filename, fmt.Sprintf("%s/release/%s/%d/%s.gz", dingDataDir, r.Name, b.ID, path.Base(res.Filename)))
		}

		release = b
	})
	events <- EventBuild{release}
	return
}

func _fileCopy(src, dst string) {
	err := os.MkdirAll(path.Dir(dst), 0777)
	_checkf(err, "making directory for copying result file")
	sf, err := os.Open(src)
	_checkf(err, "open result file")
	defer sf.Close()
	df, err := os.Create(dst)
	_checkf(err, "creating destination result file")
	gzw := gzip.NewWriter(df)
	defer func() {
		xerr := func(err1, err2 error) error {
			if err1 == nil {
				return err2
			}
			return err1
		}
		err = xerr(err, gzw.Close())
		err = xerr(err, df.Close())
		if err != nil {
			os.Remove(dst)
			_checkf(err, "installing result file")
		}
	}()
	_, err = io.Copy(gzw, sf)
	_checkf(err, "copying result file to destination")
}

// RepoBuilds is a repository and its recent builds, per branch.
type RepoBuilds struct {
	Repo   Repo
	Builds []Build // Field Steps is cleared to reduce data transferred.
}

// RepoBuilds returns all repositories and recent build info for "active" branches.
// A branch is active if its name is "master" or "main" (for git), "default" (for hg), or
// "develop", or if the last build was less than 4 weeks ago. The most recent
// build is returned.
func (Ding) RepoBuilds(ctx context.Context, password string) (rb []RepoBuilds) {
	_checkPassword(password)

	_dbread(ctx, func(tx *bstore.Tx) {
		repos, err := bstore.QueryTx[Repo](tx).List()
		_checkf(err, "list repositories")

		// repo name -> branch name -> build
		repoBuilds := map[string]map[string]Build{}
		start := time.Now().Add(-4 * 7 * 24 * time.Hour)
		err = bstore.QueryTx[Build](tx).SortDesc("ID").ForEach(func(b Build) error {
			if _, ok := repoBuilds[b.RepoName][b.Branch]; ok {
				return nil
			}
			if b.Start != nil && b.Start.Before(start) && !slices.Contains([]string{"main", "master", "default", "develop"}, b.Branch) {
				return nil
			}
			if _, ok := repoBuilds[b.RepoName]; !ok {
				repoBuilds[b.RepoName] = map[string]Build{}
			}
			b.Steps = nil
			repoBuilds[b.RepoName][b.Branch] = b
			return nil
		})
		_checkf(err, "gathering repository builds")

		rb = make([]RepoBuilds, len(repos))
		for i, r := range repos {
			rb[i].Repo = r
			rb[i].Builds = slices.Collect(maps.Values(repoBuilds[r.Name]))
			builds := rb[i].Builds
			sort.Slice(builds, func(i, j int) bool {
				a, b := builds[i], builds[j]
				return a.Created.After(b.Created)
			})
		}
		sort.Slice(rb, func(i, j int) bool {
			a, b := rb[i], rb[j]
			ba, bb := a.Builds, b.Builds
			if len(ba) == 0 && len(bb) == 0 {
				return a.Repo.Name < b.Repo.Name
			} else if len(ba) == 0 {
				return false
			} else if len(bb) == 0 {
				return true
			}
			return ba[0].Created.After(bb[0].Created)
		})
	})
	return
}

// Repo returns the named repository.
func (Ding) Repo(ctx context.Context, password, repoName string) (repo Repo) {
	_checkPassword(password)

	_dbread(ctx, func(tx *bstore.Tx) {
		repo = _repo(tx, repoName)
	})
	return
}

// Builds returns builds for a repo.
//
// The Steps field of builds is cleared for transfer size.
func (Ding) Builds(ctx context.Context, password, repoName string) (builds []Build) {
	_checkPassword(password)

	_dbread(ctx, func(tx *bstore.Tx) {
		repo := _repo(tx, repoName)
		var err error
		builds, err = bstore.QueryTx[Build](tx).FilterNonzero(Build{RepoName: repo.Name}).SortDesc("Created").List()
		_checkf(err, "fetching builds")
		for i := range builds {
			builds[i].Steps = nil
		}
	})
	return
}

func _checkRepo(repo Repo) {
	if repo.VCS != VCSCommand && repo.DefaultBranch == "" {
		_userError("DefaultBranch path cannot be empty")
	}
	if repo.CheckoutPath == "" {
		_userError("Checkout path cannot be empty")
	}
	if strings.HasPrefix(repo.CheckoutPath, "/") || strings.HasSuffix(repo.CheckoutPath, "/") {
		_userError("Checkout path cannot start or end with a slash")
	}
}

func _assignRepoUID(tx *bstore.Tx) (uid uint32) {
	uid = config.IsolateBuilds.UIDEnd - 1
	err := bstore.QueryTx[Repo](tx).ForEach(func(r Repo) error {
		if r.UID != nil && *r.UID < uid {
			uid = *r.UID
		}
		return nil
	})
	_checkf(err, "fetching last assigned repo uid from database")
	uid--
	return
}

var secretRand = newChaCha8Rand()

func newChaCha8Rand() *mathrand.Rand {
	var seed [32]byte
	_, err := cryptorand.Read(seed[:])
	if err != nil {
		panic(err)
	}
	return mathrand.New(mathrand.NewChaCha8(seed))
}

func genSecret() string {
	var r string
	const chars = "abcdefghijklmnopqrstuwvxyzABCDEFGHIJKLMNOPQRSTUWVXYZ0123456789"
	for i := 0; i < 12; i++ {
		r += string(chars[secretRand.IntN(len(chars))])
	}
	return r
}

// RepoCreate creates a new repository.
// If repo.UID is not null, a unique uid is assigned.
func (Ding) RepoCreate(ctx context.Context, password string, repo Repo) (r Repo) {
	_checkPassword(password)
	_checkRepo(repo)

	_dbwrite(ctx, func(tx *bstore.Tx) {
		var uid *uint32
		if repo.UID != nil {
			v := _assignRepoUID(tx)
			uid = &v
		}

		repo.UID = uid
		repo.HomeDiskUsage = 0
		repo.WebhookSecret = genSecret()
		err := tx.Insert(&repo)
		_checkf(err, "inserting repository in database")
		r = repo

		events <- EventRepo{r}
	})
	return
}

// RepoSave changes a repository.
func (Ding) RepoSave(ctx context.Context, password string, repo Repo) (r Repo) {
	_checkPassword(password)
	_checkRepo(repo)

	_dbwrite(ctx, func(tx *bstore.Tx) {
		r = _repo(tx, repo.Name)

		var uid *uint32
		if r.UID == nil && repo.UID != nil {
			v := _assignRepoUID(tx)
			uid = &v
		} else if repo.UID != nil {
			uid = r.UID
		}

		r.Name = repo.Name
		r.VCS = repo.VCS
		r.Origin = repo.Origin
		r.DefaultBranch = repo.DefaultBranch
		r.CheckoutPath = repo.CheckoutPath
		r.UID = uid
		r.BuildScript = repo.BuildScript
		r.NotifyEmailAddrs = repo.NotifyEmailAddrs
		r.Bubblewrap = repo.Bubblewrap
		r.BubblewrapNoNet = repo.BubblewrapNoNet
		r.BuildOnUpdatedToolchain = repo.BuildOnUpdatedToolchain
		err := tx.Update(&r)
		_checkf(err, "updating repo in database")
		r = _repo(tx, repo.Name)

		events <- EventRepo{r}
	})
	return
}

// RepoClearHomedir removes the home directory this repository shares across
// builds.
func (Ding) RepoClearHomedir(ctx context.Context, password, repoName string) {
	_checkPassword(password)

	var r Repo
	_dbread(ctx, func(tx *bstore.Tx) {
		r = _repo(tx, repoName)
		if r.UID == nil {
			_userError("repo does not share home directory across builds")
		}
	})

	msg := msg{RemoveSharedHome: &msgRemoveSharedHome{repoName}}
	err := requestPrivileged(msg)
	_checkf(err, "privileged RemoveSharedHome")

	_dbwrite(context.Background(), func(tx *bstore.Tx) {
		r = _repo(tx, repoName)
		r.HomeDiskUsage = 0
		err := tx.Update(&r)
		_checkf(err, "updating repo home disk usage in database")
	})
}

// ClearRepoHomedirs removes the home directory of all repositories.
func (Ding) ClearRepoHomedirs(ctx context.Context, password string) {
	_checkPassword(password)

	repos, err := bstore.QueryDB[Repo](ctx, database).FilterFn(func(r Repo) bool { return r.UID != nil }).List()
	_checkf(err, "fetching repo names to clear from database")

	for _, repo := range repos {
		msg := msg{RemoveSharedHome: &msgRemoveSharedHome{repo.Name}}
		err := requestPrivileged(msg)
		_checkf(err, "privileged RemoveSharedHome")

		_dbwrite(context.Background(), func(tx *bstore.Tx) {
			r := _repo(tx, repo.Name)
			r.HomeDiskUsage = 0
			err = tx.Update(&r)
			_checkf(err, "update repo home disk usage")
		})
	}
}

// RepoRemove removes a repository and all its builds.
func (Ding) RepoRemove(ctx context.Context, password, repoName string) {
	_checkPassword(password)

	_dbwrite(ctx, func(tx *bstore.Tx) {
		repo := _repo(tx, repoName)

		_, err := bstore.QueryTx[Build](tx).FilterNonzero(Build{RepoName: repo.Name}).Delete()
		_checkf(err, "deleting builds from database")

		err = tx.Delete(repo)
		_checkf(err, "removing repo from database")
	})
	events <- EventRemoveRepo{repoName}

	err := requestPrivileged(msg{RemoveRepo: &msgRemoveRepo{repoName}})
	_checkf(err, "removing repo files")

	err = os.RemoveAll(fmt.Sprintf("%s/release/%s", dingDataDir, repoName))
	_checkf(err, "removing release directory")
}

// Build returns the build, including steps output.
func (Ding) Build(ctx context.Context, password, repoName string, buildID int32) (b Build) {
	_checkPassword(password)

	_dbread(ctx, func(tx *bstore.Tx) {
		_, b = _build(tx, repoName, buildID)
	})

	if b.Finish == nil {
		b.Steps = _buildSteps(b)
	}

	return
}

// BuildRemove removes a build completely. Both from database and all local files.
func (Ding) BuildRemove(ctx context.Context, password string, buildID int32) {
	_checkPassword(password)

	var repoName string
	_dbwrite(ctx, func(tx *bstore.Tx) {
		b := Build{ID: buildID}
		err := tx.Get(&b)
		_checkf(err, "fetching repo name from database")

		if b.Released != nil {
			_userError("Build has been released, cannot be removed")
		}

		r := Repo{Name: b.RepoName}
		err = tx.Get(&r)
		_checkf(err, "get repo")
		repoName = r.Name

		_removeBuild(tx, r.Name, b.ID)
	})
	events <- EventRemoveBuild{repoName, buildID}
}

// BuildCleanupBuilddir cleans up (removes) a build directory.
// This does not remove the build itself from the database.
func (Ding) BuildCleanupBuilddir(ctx context.Context, password, repoName string, buildID int32) (build Build) {
	_checkPassword(password)

	_dbwrite(ctx, func(tx *bstore.Tx) {
		_, b := _build(tx, repoName, buildID)
		if b.BuilddirRemoved {
			_userError("Builddir already removed")
		}

		_removeBuildDir(b)

		b.BuilddirRemoved = true
		err := tx.Update(&b)
		_checkf(err, "marking builddir as removed")

		build = b
	})
	events <- EventBuild{build}
	return
}

// GoToolchains lists the active current, previous and next versions of the Go
// toolchain, as symlinked in $DING_TOOLCHAINDIR.
type GoToolchains struct {
	Go     string
	GoPrev string
	GoNext string
}

// GoToolchainsListInstalled returns the installed Go toolchains (eg "go1.13.8",
// "go1.14") in GoToolchainDir, and current "active" versions with a shortname, eg
// "go" as "go1.14", "go-prev" as "go1.13.8" and "go-next" as "go1.23rc1".
func (Ding) GoToolchainsListInstalled(ctx context.Context, password string) (installed []string, active GoToolchains) {
	_checkPassword(password)

	_checkGoToolchainDir()

	files, err := os.ReadDir(config.GoToolchainDir)
	_checkf(err, "listing files in go toolchain dir")
	for _, f := range files {
		if f.IsDir() && strings.HasPrefix(f.Name(), "go") {
			installed = append(installed, f.Name())
		}
	}

	active.Go, _ = os.Readlink(path.Join(config.GoToolchainDir, "go"))
	active.GoPrev, _ = os.Readlink(path.Join(config.GoToolchainDir, "go-prev"))
	active.GoNext, _ = os.Readlink(path.Join(config.GoToolchainDir, "go-next"))

	return
}

var releasedCache struct {
	sync.Mutex
	expires  time.Time
	released []string
}

// GoToolchainsListReleased returns all known released Go toolchains available at
// golang.org/dl/, eg "go1.13.8", "go1.14".
func (Ding) GoToolchainsListReleased(ctx context.Context, password string) (released []string) {
	_checkPassword(password)

	releasedCache.Lock()
	defer releasedCache.Unlock()

	if time.Now().After(releasedCache.expires) {
		releases, err := goreleases.ListAll()
		_checkf(err, "fetching list of all released go toolchains")
		releasedCache.released = []string{}
		for _, rel := range releases {
			releasedCache.released = append(releasedCache.released, rel.Version)
		}
		releasedCache.expires = time.Now().Add(time.Minute)
	}
	released = releasedCache.released
	return
}

// GoToolchainInstall downloads, verifies and extracts the release Go toolchain
// represented by goversion (eg "go1.13.8", "go1.14") into the GoToolchainDir, and
// optionally "activates" the version under shortname ("go", "go-prev", "go-next", ""; empty
// string does nothing).
func (Ding) GoToolchainInstall(ctx context.Context, password, goversion, shortname string) {
	_checkPassword(password)

	_checkGoToolchainDir()
	if !validGoversion(goversion) {
		_userError("bad goversion")
	}

	switch shortname {
	case "go", "go-prev", "go-next", "":
	default:
		_userError("invalid shortname")
	}

	// Check goversion isn't already installed.
	versionDst := path.Join(config.GoToolchainDir, goversion)
	_, err := os.Stat(versionDst)
	if err == nil {
		_userError("already installed")
	}

	releases, err := goreleases.ListAll()
	_checkf(err, "fetching list of all released go toolchains")

	rel := _findRelease(releases, goversion)
	file, err := goreleases.FindFile(rel, runtime.GOOS, runtime.GOARCH, "archive")
	_checkf(err, "finding file for running os and arch")

	msg := msg{InstallGoToolchain: &msgInstallGoToolchain{file, shortname}}
	err = requestPrivileged(msg)
	_checkf(err, "install go toolchain")
}

// GoToolchainRemove removes a toolchain from the go toolchain dir.
// It also removes shortname symlinks to this toolchain if they exists.
func (Ding) GoToolchainRemove(ctx context.Context, password, goversion string) {
	_checkPassword(password)

	_checkGoToolchainDir()
	if !validGoversion(goversion) {
		_userError("bad goversion")
	}

	msg := msg{RemoveGoToolchain: &msgRemoveGoToolchain{goversion}}
	err := requestPrivileged(msg)
	_checkf(err, "removing go toolchain")
}

// GoToolchainActivate activates goversion (eg "go1.13.8", "go1.14") under the name
// shortname ("go", "go-prev" or "go-next"), by creating a symlink in the GoToolchainDir.
func (Ding) GoToolchainActivate(ctx context.Context, password, goversion, shortname string) {
	_checkPassword(password)

	_checkGoToolchainDir()
	if !validGoversion(goversion) {
		_userError("bad goversion")
	}

	switch shortname {
	case "go", "go-prev", "go-next":
		msg := msg{ActivateGoToolchain: &msgActivateGoToolchain{goversion, shortname}}
		err := requestPrivileged(msg)
		_checkf(err, "removing go toolchain")
	default:
		_userError("invalid shortname")
	}
}

// GoToolchainAutomatic looks up the latest released Go toolchains, and installs
// the current and previous releases, and the next (release candidate) if present.
// Then it starts low-prio builds for all repositories that have opted in to
// automatic building on new Go toolchains.
func (Ding) GoToolchainAutomatic(ctx context.Context, password string) (updated bool) {
	_checkPassword(password)

	msg := msg{AutomaticGoToolchain: &msgAutomaticGoToolchain{}}
	err := requestPrivileged(msg)
	if err != nil && err.Error() == "updated" {
		updated = true
		err = nil
	}
	_checkf(err, "updating automatic go toolchains")
	if updated {
		err := scheduleLowPrioBuilds(ctx, true)
		_checkf(err, "scheduling low prio builds after updated toolchains")
	}
	return
}

func _checkGoToolchainDir() {
	if config.GoToolchainDir == "" {
		_userError("GoToolchainDir not configured")
	}
}

func _findRelease(releases []goreleases.Release, goversion string) goreleases.Release {
	for _, rel := range releases {
		if rel.Version == goversion {
			return rel
		}
	}
	_userError("version not found")
	return goreleases.Release{}
}

// LogLevel indicates the severity of a log message.
type LogLevel string

// LogLevels for setting the active log level.
const (
	LogDebug LogLevel = "debug"
	LogInfo  LogLevel = "info"
	LogWarn  LogLevel = "warn"
	LogError LogLevel = "error"
)

// LogLevel returns the current log level.
func (Ding) LogLevel(ctx context.Context, password string) LogLevel {
	_checkPassword(password)

	switch loglevel.Level() {
	case slog.LevelDebug:
		return LogDebug
	case slog.LevelInfo:
		return LogInfo
	case slog.LevelWarn:
		return LogWarn
	case slog.LevelError:
		return LogError
	}
	return LogLevel("")
}

// LogLevelSet sets a new log level.
func (Ding) LogLevelSet(ctx context.Context, password string, level LogLevel) {
	_checkPassword(password)

	var nlevel slog.Level
	switch level {
	case LogDebug:
		nlevel = slog.LevelDebug
	case LogInfo:
		nlevel = slog.LevelInfo
	case LogWarn:
		nlevel = slog.LevelWarn
	case LogError:
		nlevel = slog.LevelError
	default:
		_userError(fmt.Sprintf("unknown loglevel %q", level))
	}

	// Propagate to privileged process.
	err := requestPrivileged(msg{LogLevelSet: &msgLogLevelSet{LogLevel: nlevel}})
	_checkf(err, "setting log level")
	loglevel.Set(nlevel)
}

// Settings returns the runtime settings.
func (Ding) Settings(ctx context.Context, password string) (isolationEnabled bool, mailEnabled bool, settings Settings) {
	_checkPassword(password)

	settings.ID = 1
	err := database.Get(ctx, &settings)
	_checkf(err, "get settings")
	isolationEnabled = config.IsolateBuilds.Enabled
	mailEnabled = config.Mail.Enabled
	return
}

// SettingsSave saves the runtime settings.
func (Ding) SettingsSave(ctx context.Context, password string, settings Settings) {
	_checkPassword(password)
	err := database.Update(ctx, &settings)
	_checkf(err, "update settings")
}

// Version returns the ding version this instance is running.
func (Ding) Version(ctx context.Context, password string) (dingversion, goos, goarch, goversion string, haveBubblewrap bool) {
	_checkPassword(password)

	// Check if bwrap is present.
	cmd := exec.CommandContext(ctx, "which", "bwrap")
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	err := cmd.Run()
	haveBubblewrap = err == nil

	return version, runtime.GOOS, runtime.GOARCH, runtime.Version(), haveBubblewrap
}
