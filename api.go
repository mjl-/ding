package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"path"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mjl-/bstore"
	"github.com/mjl-/goreleases"
	"github.com/mjl-/sherpa"
)

// The Ding API lets you compile git branches, build binaries, run tests, and publish binaries.
type Ding struct {
	SSE SSE `sherpa:"Server-Sent Events"`
}

// Status checks the health of the application.
// If backend connectivity is broken, this sherpa call results in a 500 internal server error. Useful for monitoring tools.
func (Ding) Status(ctx context.Context) {
	type what int
	const (
		filesystem what = iota
		timer
	)

	type done struct {
		what  what
		error bool
	}

	errors := make(chan done, 2)

	go func() {
		defer os.Remove(dingDataDir + "/test")
		f, err := os.Create(dingDataDir + "/test")
		if err == nil {
			err = f.Close()
		}
		if err != nil {
			log.Printf("status: file system unavailable: %s", err)
			errors <- done{filesystem, true}
			return
		}
		errors <- done{filesystem, false}
	}()

	timeout := time.AfterFunc(time.Second*5, func() {
		log.Println("status: timeout for fs checks")
		errors <- done{timer, true}
	})

	statusError := func(msg string) {
		log.Println("status:", msg)
		panic(&sherpa.InternalServerError{Code: "serverError", Message: msg})
	}

	fs := false
	for !fs {
		done := <-errors
		if !done.error {
			switch done.what {
			case filesystem:
				fs = true
			default:
				_serverError("status: internal error")
			}
			continue
		}

		timeout.Stop()
		switch done.what {
		case filesystem:
			statusError("filesystem unavailable")
		case timer:
			if !fs {
				statusError("timeout for filesystem")
			}
		default:
			_serverError("status: missing case")
		}
	}
	timeout.Stop()
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

// CreateBuild builds a specific commit in the background, returning immediately.
// `Commit` can be empty, in which case the origin is cloned and the checked out commit is looked up.
func (Ding) CreateBuild(ctx context.Context, password, repoName, branch, commit string) Build {
	_checkPassword(password)
	return _createBuildPrio(ctx, repoName, branch, commit, false)
}

// CreateBuildLowPrio creates a build, but with low priority.
// Low priority builds are executed after regular builds. And only one low priority build is running over all repo's.
func (Ding) CreateBuildLowPrio(ctx context.Context, password, repoName, branch, commit string) Build {
	_checkPassword(password)
	return _createBuildPrio(ctx, repoName, branch, commit, true)
}

func _createBuildPrio(ctx context.Context, repoName, branch, commit string, lowPrio bool) Build {
	if branch == "" {
		_userError("Branch cannot be empty")
	}

	repo, build, buildDir := _prepareBuild(ctx, repoName, branch, commit, lowPrio)
	go func() {
		defer func() {
			if x := recover(); x != nil {
				log.Println("build:", x)
			}
		}()
		_doBuild(context.Background(), repo, build, buildDir)
	}()
	return build
}

// CreateLowPrioBuilds creates low priority builds for each repository, for the default branch.
func (Ding) CreateLowPrioBuilds(ctx context.Context, password string) {
	_checkPassword(password)

	repos, err := bstore.QueryDB[Repo](ctx, database).List()
	_checkf(err, "fetching repo names from database")

	lowPrio := true
	commit := ""

	builds := make([]Build, len(repos))
	buildDirs := make([]string, len(repos))
	for i, repo := range repos {
		_, build, buildDir := _prepareBuild(ctx, repo.Name, repo.DefaultBranch, commit, lowPrio)
		builds[i] = build
		buildDirs[i] = buildDir
	}

	for i, repo := range repos {
		build := builds[i]
		buildDir := buildDirs[i]
		go func() {
			defer func() {
				if x := recover(); x != nil {
					log.Println("lowprio build:", x)
				}
			}()
			_doBuild(context.Background(), repo, build, buildDir)
		}()
	}
}

// CancelBuild cancels a currently running build.
func (Ding) CancelBuild(ctx context.Context, password, repoName string, buildID int32) {
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
	go requestPrivileged(cancelMsg)
}

// CreateRelease release a build.
func (Ding) CreateRelease(ctx context.Context, password, repoName string, buildID int32) (release Build) {
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
	events <- EventBuild{release.RepoName, release}
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
	Repo   Repo    `json:"repo"`
	Builds []Build `json:"builds"`
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
func (Ding) Builds(ctx context.Context, password, repoName string) (builds []Build) {
	_checkPassword(password)

	_dbread(ctx, func(tx *bstore.Tx) {
		repo := _repo(tx, repoName)
		var err error
		builds, err = bstore.QueryTx[Build](tx).FilterNonzero(Build{RepoName: repo.Name}).SortDesc("Created").List()
		_checkf(err, "fetching builds")
	})
	return
}

func _checkRepo(repo Repo) {
	if repo.DefaultBranch == "" {
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

// CreateRepo creates a new repository.
// If repo.UID is not null, a unique uid is assigned.
func (Ding) CreateRepo(ctx context.Context, password string, repo Repo) (r Repo) {
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
		err := tx.Insert(&repo)
		_checkf(err, "inserting repository in database")
		r = repo

		events <- EventRepo{r}
	})
	return
}

// SaveRepo changes a repository.
func (Ding) SaveRepo(ctx context.Context, password string, repo Repo) (r Repo) {
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
		err := tx.Update(&r)
		_checkf(err, "updating repo in database")
		r = _repo(tx, repo.Name)

		events <- EventRepo{r}
	})
	return
}

// ClearRepoHomedir removes the home directory this repository shares across builds.
func (Ding) ClearRepoHomedir(ctx context.Context, password, repoName string) {
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

// RemoveRepo removes a repository and all its builds.
func (Ding) RemoveRepo(ctx context.Context, password, repoName string) {
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

func parseInt(s string) int64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	_checkf(err, "parsing integer")
	return v
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
			Name:   stepName,
			Output: readFileLax(base + ".output"),
			Nsec:   parseInt(readFileLax(base + ".nsec")),
		})
		if stepName == b.Status {
			break
		}
	}
	return
}

// Build returns the build and steps of the requested build.
func (Ding) Build(ctx context.Context, password, repoName string, buildID int32) (b Build) {
	_checkPassword(password)

	_dbread(ctx, func(tx *bstore.Tx) {
		_, b = _build(tx, repoName, buildID)
	})
	return
}

// Release fetches the build config and results for a release.
func (Ding) Release(ctx context.Context, password, repoName string, buildID int32) (release Build) {
	_checkPassword(password)

	_dbread(ctx, func(tx *bstore.Tx) {
		_, b := _build(tx, repoName, buildID)
		if b.Released == nil {
			_userError("Build not released")
		}
		release = b
	})
	return
}

// RemoveBuild removes a build completely. Both from database and all local files.
func (Ding) RemoveBuild(ctx context.Context, password string, buildID int32) {
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

// CleanupBuilddir cleans up (removes) a build directory.
// This does not remove the build itself from the database.
func (Ding) CleanupBuilddir(ctx context.Context, password, repoName string, buildID int32) (build Build) {
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
	events <- EventBuild{repoName, build}
	return
}

// ListInstalledGoToolchains returns the installed Go toolchains (eg "go1.13.8",
// "go1.14") in GoToolchainDir, and current "active" versions with a shortname, eg
// "go" as "go1.14" and "go-prev" as "go1.13.8".
func (Ding) ListInstalledGoToolchains(ctx context.Context, password string) (installed []string, active map[string]string) {
	_checkPassword(password)

	_checkGoToolchainDir()

	files, err := os.ReadDir(config.GoToolchainDir)
	_checkf(err, "listing files in go toolchain dir")

	active = map[string]string{}
	for _, f := range files {
		if !strings.HasPrefix(f.Name(), "go") {
			continue
		}
		if f.IsDir() {
			installed = append(installed, f.Name())
			continue
		}
		if f.Type()&os.ModeSymlink != 0 {
			switch f.Name() {
			case "go", "go-prev":
			default:
				continue
			}
			goversion, err := os.Readlink(path.Join(config.GoToolchainDir, f.Name()))
			_checkf(err, "reading go symlink for active go toolchain")
			active[f.Name()] = goversion
		}
	}
	return
}

var releasedCache struct {
	sync.Mutex
	expires  time.Time
	released []string
}

// ListReleasedGoToolchains returns all known released Go toolchains available at
// golang.org/dl/, eg "go1.13.8", "go1.14".
func (Ding) ListReleasedGoToolchains(ctx context.Context, password string) (released []string) {
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

// InstallGoToolchain downloads, verifies and extracts the release Go toolchain
// represented by goversion (eg "go1.13.8", "go1.14") into the GoToolchainDir, and
// optionally "activates" the version under shortname ("go", "go-prev", ""; empty
// string does nothing).
func (Ding) InstallGoToolchain(ctx context.Context, password, goversion, shortname string) {
	_checkPassword(password)

	_checkGoToolchainDir()
	if !validGoversion(goversion) {
		_userError("bad goversion")
	}

	switch shortname {
	case "go", "go-prev", "":
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

// RemoveGoToolchain removes a toolchain from go toolchain dir.
// It does not remove a shortname symlink to this toolchain if it exists.
func (Ding) RemoveGoToolchain(ctx context.Context, password, goversion string) {
	_checkPassword(password)

	_checkGoToolchainDir()
	if !validGoversion(goversion) {
		_userError("bad goversion")
	}

	msg := msg{RemoveGoToolchain: &msgRemoveGoToolchain{goversion}}
	err := requestPrivileged(msg)
	_checkf(err, "removing go toolchain")
}

// ActivateGoToolchain activates goversion (eg "go1.13.8", "go1.14") under the name
// shortname ("go" or "go-prev"), by creating a symlink in the GoToolchainDir.
func (Ding) ActivateGoToolchain(ctx context.Context, password, goversion, shortname string) {
	_checkPassword(password)

	_checkGoToolchainDir()
	if !validGoversion(goversion) {
		_userError("bad goversion")
	}

	switch shortname {
	case "go", "go-prev":
		msg := msg{ActivateGoToolchain: &msgActivateGoToolchain{goversion, shortname}}
		err := requestPrivileged(msg)
		_checkf(err, "removing go toolchain")
	default:
		_userError("invalid shortname")
	}
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
