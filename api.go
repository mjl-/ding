package main

import (
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mjl-/goreleases"
	"github.com/mjl-/sherpa"
)

var (
	stepNames = []string{
		"clone",
		"build",
	}
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
		xdatabase
		timer
	)

	type done struct {
		what  what
		error bool
	}

	errors := make(chan done, 3)

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

	go func() {
		var one int
		err := database.QueryRow("select 1").Scan(&one)
		if err != nil {
			log.Printf("status: database unavailable: %s", err)
			errors <- done{xdatabase, true}
			return
		}
		errors <- done{xdatabase, false}
	}()

	timeout := time.AfterFunc(time.Second*5, func() {
		log.Println("status: timeout for db or fs checks")
		errors <- done{timer, true}
	})

	statusError := func(msg string) {
		log.Println("status:", msg)
		panic(&sherpa.InternalServerError{Code: "serverError", Message: msg})
	}

	db := false
	fs := false
	for !db || !fs {
		done := <-errors
		if !done.error {
			switch done.what {
			case filesystem:
				fs = true
			case xdatabase:
				db = true
			default:
				serverError("status: internal error")
			}
			continue
		}

		timeout.Stop()
		switch done.what {
		case filesystem:
			statusError("filesystem unavailable")
		case xdatabase:
			statusError("database unavailable")
		case timer:
			if !db && !fs {
				statusError("timeout for both filesystem and database")
			}
			if !db {
				statusError("timeout for database")
			}
			if !fs {
				statusError("timeout for filesystem")
			}
		default:
			serverError("status: missing case")
		}
	}
	timeout.Stop()
}

func _repo(tx *sql.Tx, repoName string) (r Repo) {
	q := `select row_to_json(repo.*) from repo where name=$1`
	sherpaCheckRow(tx.QueryRow(q, repoName), &r, "fetching repo")
	return
}

func _build(tx *sql.Tx, repoName string, id int32) (b Build) {
	q := `select row_to_json(bwr.*) from build_with_result bwr where id = $1`
	sherpaCheckRow(tx.QueryRow(q, id), &b, "fetching build")
	fillBuild(repoName, &b)
	return
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
	return createBuildPrio(ctx, repoName, branch, commit, false)
}

// CreateBuildLowPrio creates a build, but with low priority.
// Low priority builds are executed after regular builds. And only one low priority build is running over all repo's.
func (Ding) CreateBuildLowPrio(ctx context.Context, password, repoName, branch, commit string) Build {
	_checkPassword(password)
	return createBuildPrio(ctx, repoName, branch, commit, true)
}

func createBuildPrio(ctx context.Context, repoName, branch, commit string, lowPrio bool) Build {
	if branch == "" {
		userError("Branch cannot be empty.")
	}

	repo, build, buildDir := _prepareBuild(ctx, repoName, branch, commit, lowPrio)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				if serr, ok := err.(*sherpa.Error); ok {
					if serr.Code != "userError" {
						log.Println("background build failed:", serr.Message)
					}
				} else {
					panic(err)
				}
			}
		}()
		doBuild(context.Background(), repo, build, buildDir)
	}()
	return build
}

// CreateLowPrioBuilds creates low priority builds for each repository, for the default branch.
func (Ding) CreateLowPrioBuilds(ctx context.Context, password string) {
	_checkPassword(password)

	var repos []Repo
	transact(ctx, func(tx *sql.Tx) {
		q := `select coalesce(json_agg(repo.* order by id desc), '[]') from repo where uid is not null`
		sherpaCheckRow(tx.QueryRow(q), &repos, "fetching repo names to clear from database")
	})

	lowPrio := true
	commit := ""

	for _, repo := range repos {
		repo, build, buildDir := _prepareBuild(ctx, repo.Name, repo.DefaultBranch, commit, lowPrio)
		go func() {
			defer func() {
				if err := recover(); err != nil {
					if serr, ok := err.(*sherpa.Error); ok {
						if serr.Code != "userError" {
							log.Println("background build failed:", serr.Message)
						}
					} else {
						panic(err)
					}
				}
			}()
			doBuild(context.Background(), repo, build, buildDir)
		}()
	}
}

// CancelBuild cancels a currently running build.
func (Ding) CancelBuild(ctx context.Context, password, repoName string, buildID int32) {
	_checkPassword(password)

	transact(ctx, func(tx *sql.Tx) {
		repo := _repo(tx, repoName)

		build := _build(tx, repo.Name, buildID)
		if build.Finish != nil {
			userError("Build has already finished")
		}

		q := `update build set finish=now(), status='cancelled' where id=$1 and finish is null`
		_, err := tx.Exec(q, buildID)
		sherpaCheck(err, "marking build as cancelled in database")
	})

	// Cancel any commands in the http-serve process, like cloning.
	buildIDCommandCancel(buildID)

	// And cancel the actual build command controlled by the serve process.
	cancelMsg := msg{CancelCommand: &msgCancelCommand{buildID}}
	go requestPrivileged(cancelMsg)
}

func toJSON(v interface{}) string {
	buf, err := json.Marshal(v)
	sherpaCheck(err, "encoding to json")
	return string(buf)
}

// CreateRelease release a build.
func (Ding) CreateRelease(ctx context.Context, password, repoName string, buildID int32) (build Build) {
	_checkPassword(password)

	transact(ctx, func(tx *sql.Tx) {
		repo := _repo(tx, repoName)

		build = _build(tx, repo.Name, buildID)
		if build.Finish == nil {
			userError("Build has not finished yet")
		}
		if build.Status != "success" {
			userError("Build was not successful")
		}

		br := _buildResult(repo.Name, build)
		steps := toJSON(br.Steps)

		qrel := `insert into release (build_id, time, build_script, steps) values ($1, now(), $2, $3::json) returning build_id`
		err := tx.QueryRow(qrel, build.ID, br.BuildScript, steps).Scan(&build.ID)
		sherpaCheck(err, "inserting release into database")

		qup := `update build set released=now() where id=$1 returning id`
		err = tx.QueryRow(qup, build.ID).Scan(&build.ID)
		sherpaCheck(err, "marking build as released in database")

		var filenames []string
		q := `select coalesce(json_agg(result.filename), '[]') from result where build_id=$1`
		sherpaCheckRow(tx.QueryRow(q, build.ID), &filenames, "fetching build results from database")
		checkoutDir := fmt.Sprintf("%s/build/%s/%d/checkout/%s", dingDataDir, repo.Name, build.ID, repo.CheckoutPath)
		for _, filename := range filenames {
			fileCopy(checkoutDir+"/"+filename, fmt.Sprintf("%s/release/%s/%d/%s.gz", dingDataDir, repo.Name, build.ID, path.Base(filename)))
		}

		events <- EventBuild{repo.Name, _build(tx, repo.Name, buildID)}
	})
	return
}

func fileCopy(src, dst string) {
	err := os.MkdirAll(path.Dir(dst), 0777)
	sherpaCheck(err, "making directory for copying result file")
	sf, err := os.Open(src)
	sherpaCheck(err, "open result file")
	defer sf.Close()
	df, err := os.Create(dst)
	sherpaCheck(err, "creating destination result file")
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
			sherpaCheck(err, "installing result file")
		}
	}()
	_, err = io.Copy(gzw, sf)
	sherpaCheck(err, "copying result file to destination")
}

// RepoBuilds returns all repositories and recent build info for "active" branches.
// A branch is active if its name is "master" or "main" (for git), "default" (for hg), or
// "develop", or if the last build was less than 4 weeks ago. The most recent
// completed build is returned, and optionally the first build in progress.
func (Ding) RepoBuilds(ctx context.Context, password string) (rb []RepoBuilds) {
	_checkPassword(password)

	q := `
		with repo_branch_builds as (
				select *
				from build_with_result
				where id in (
					select max(id) as id
					from build
					where true
						and (branch in ('main', 'master', 'default', 'develop') or start > now() - interval '4 weeks')
						and build.finish is not null
					group by repo_id, branch
				)
			union all
				select *
				from build_with_result
				where id in (
					select min(id) as id
					from build
					where true
						and (branch in ('main', 'master', 'default', 'develop') or start > now() - interval '4 weeks')
						and build.finish is null
					group by repo_id, branch
				)
		)
		select coalesce(json_agg(repobuilds.*), '[]')
		from (
			select row_to_json(repo.*) as repo, array_remove(array_agg(rbb.*), null) as builds
			from repo
			left join repo_branch_builds rbb on repo.id = rbb.repo_id
			group by repo.id
		) repobuilds
	`
	sherpaCheckRow(database.QueryRowContext(ctx, q), &rb, "fetching repobuilds")
	for _, e := range rb {
		for i, b := range e.Builds {
			fillBuild(e.Repo.Name, &b)
			e.Builds[i] = b
		}
	}
	return
}

// Repo returns the named repository.
func (Ding) Repo(ctx context.Context, password, repoName string) (repo Repo) {
	_checkPassword(password)

	transact(ctx, func(tx *sql.Tx) {
		repo = _repo(tx, repoName)
	})
	return
}

// Builds returns builds for a repo.
func (Ding) Builds(ctx context.Context, password, repoName string) (builds []Build) {
	_checkPassword(password)

	q := `select coalesce(json_agg(bwr.* order by start desc), '[]') from build_with_result bwr join repo on bwr.repo_id = repo.id where repo.name=$1`
	sherpaCheckRow(database.QueryRowContext(ctx, q, repoName), &builds, "fetching builds")
	for i, b := range builds {
		fillBuild(repoName, &b)
		builds[i] = b
	}
	return
}

func _checkRepo(repo Repo) {
	if repo.DefaultBranch == "" {
		userError("DefaultBranch path cannot be empty.")
	}
	if repo.CheckoutPath == "" {
		userError("Checkout path cannot be empty.")
	}
	if strings.HasPrefix(repo.CheckoutPath, "/") || strings.HasSuffix(repo.CheckoutPath, "/") {
		userError("Checkout path cannot start or end with a slash.")
	}
}

func _assignRepoUID(tx *sql.Tx) (uid uint32) {
	q := `select coalesce(min(uid), $1) - 1 as uid from repo`
	err := tx.QueryRow(q, config.IsolateBuilds.UIDEnd-1).Scan(&uid)
	sherpaCheck(err, "fetching last assigned repo uid from database")
	return
}

// CreateRepo creates a new repository.
// If repo.UID is not null, a unique uid is assigned.
func (Ding) CreateRepo(ctx context.Context, password string, repo Repo) (r Repo) {
	_checkPassword(password)
	_checkRepo(repo)

	transact(ctx, func(tx *sql.Tx) {
		var uid interface{}
		if repo.UID != nil {
			uid = _assignRepoUID(tx)
		}

		q := `insert into repo (name, vcs, origin, default_branch, checkout_path, uid, build_script) values ($1, $2, $3, $4, $5, $6, '') returning id`
		var id int64
		sherpaCheckRow(tx.QueryRow(q, repo.Name, repo.VCS, repo.Origin, repo.DefaultBranch, repo.CheckoutPath, uid), &id, "inserting repository in database")
		r = _repo(tx, repo.Name)

		events <- EventRepo{r}
	})
	return
}

// SaveRepo changes a repository.
func (Ding) SaveRepo(ctx context.Context, password string, repo Repo) (r Repo) {
	_checkPassword(password)
	_checkRepo(repo)

	transact(ctx, func(tx *sql.Tx) {
		r = _repo(tx, repo.Name)
		var uid interface{}
		if r.UID == nil && repo.UID != nil {
			uid = _assignRepoUID(tx)
		} else if repo.UID != nil {
			uid = *r.UID
		}

		q := `update repo set name=$1, vcs=$2, origin=$3, default_branch=$4, checkout_path=$5, uid=$6, build_script=$7 where id=$8 returning row_to_json(repo.*)`
		sherpaCheckRow(tx.QueryRow(q, repo.Name, repo.VCS, repo.Origin, repo.DefaultBranch, repo.CheckoutPath, uid, repo.BuildScript, repo.ID), &r, "updating repo in database")
		r = _repo(tx, repo.Name)

		events <- EventRepo{r}
	})
	return
}

// ClearRepoHomedir removes the home directory this repository shares across builds.
func (Ding) ClearRepoHomedir(ctx context.Context, password, repoName string) {
	_checkPassword(password)

	var r Repo
	transact(ctx, func(tx *sql.Tx) {
		r = _repo(tx, repoName)
		if r.UID == nil {
			userError("repo does not share home directory across builds")
		}
	})

	msg := msg{RemoveSharedHome: &msgRemoveSharedHome{repoName}}
	err := requestPrivileged(msg)
	sherpaCheck(err, "privileged RemoveSharedHome")

	transact(context.Background(), func(tx *sql.Tx) {
		q := `update repo set home_disk_usage=0 where id=$1 returning 1`
		var one int
		sherpaCheckRow(tx.QueryRow(q, r.ID), &one, "updating repo home disk usage in database")
	})
}

// ClearRepoHomedirs removes the home directory of all repositories.
func (Ding) ClearRepoHomedirs(ctx context.Context, password string) {
	_checkPassword(password)

	var repos []Repo
	transact(ctx, func(tx *sql.Tx) {
		q := `select coalesce(json_agg(repo.*), '[]') from repo where uid is not null`
		sherpaCheckRow(tx.QueryRow(q), &repos, "fetching repo names to clear from database")
	})

	for _, repo := range repos {
		msg := msg{RemoveSharedHome: &msgRemoveSharedHome{repo.Name}}
		err := requestPrivileged(msg)
		sherpaCheck(err, "privileged RemoveSharedHome")

		transact(context.Background(), func(tx *sql.Tx) {
			q := `update repo set home_disk_usage=0 where id=$1 returning 1`
			var one int
			sherpaCheckRow(tx.QueryRow(q, repo.ID), &one, "updating repo home disk usage in database")
		})
	}
}

// RemoveRepo removes a repository and all its builds.
func (Ding) RemoveRepo(ctx context.Context, password, repoName string) {
	_checkPassword(password)

	transact(ctx, func(tx *sql.Tx) {
		_repo(tx, repoName)

		_, err := tx.Exec(`delete from result where build_id in (select id from build where repo_id in (select id from repo where name=$1))`, repoName)
		sherpaCheck(err, "removing results from database")

		_, err = tx.Exec(`delete from build where repo_id in (select id from repo where name=$1)`, repoName)
		sherpaCheck(err, "removing builds from database")

		var id int
		sherpaCheckRow(tx.QueryRow(`delete from repo where name=$1 returning id`, repoName), &id, "removing repo from database")
	})
	events <- EventRemoveRepo{repoName}

	err := requestPrivileged(msg{RemoveRepo: &msgRemoveRepo{repoName}})
	sherpaCheck(err, "removing repo files")

	err = os.RemoveAll(fmt.Sprintf("%s/release/%s", dingDataDir, repoName))
	sherpaCheck(err, "removing release directory")
}

func parseInt(s string) int64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	sherpaCheck(err, "parsing integer")
	return v
}

func _buildResult(repoName string, build Build) (br BuildResult) {
	buildDir := fmt.Sprintf("%s/build/%s/%d/", dingDataDir, repoName, build.ID)
	br.BuildScript = readFile(buildDir + "scripts/build.sh")
	br.Steps = []Step{}

	if build.Status == "new" {
		return
	}

	outputDir := buildDir + "output/"
	for _, stepName := range stepNames {
		br.Steps = append(br.Steps, Step{
			Name:   stepName,
			Stdout: readFileLax(outputDir + stepName + ".stdout"),
			Stderr: readFileLax(outputDir + stepName + ".stderr"),
			Output: readFileLax(outputDir + stepName + ".output"),
			Nsec:   parseInt(readFileLax(outputDir + stepName + ".nsec")),
		})
		if stepName == build.Status {
			break
		}
	}
	return
}

// BuildResult returns the results of the requested build.
func (Ding) BuildResult(ctx context.Context, password, repoName string, buildID int32) (br BuildResult) {
	_checkPassword(password)

	var build Build
	transact(ctx, func(tx *sql.Tx) {
		build = _build(tx, repoName, buildID)
	})
	br = _buildResult(repoName, build)
	br.Build = build
	return
}

// Release fetches the build config and results for a release.
func (Ding) Release(ctx context.Context, password, repoName string, buildID int32) (br BuildResult) {
	_checkPassword(password)

	transact(ctx, func(tx *sql.Tx) {
		build := _build(tx, repoName, buildID)

		q := `select row_to_json(release.*) from release where build_id=$1`
		sherpaCheckRow(tx.QueryRow(q, buildID), &br, "fetching release from database")
		br.Build = build
	})
	return
}

// RemoveBuild removes a build completely. Both from database and all local files.
func (Ding) RemoveBuild(ctx context.Context, password string, buildID int32) {
	_checkPassword(password)

	var repoName string
	transact(ctx, func(tx *sql.Tx) {
		qrepo := `select to_json(repo.name) from build join repo on build.repo_id = repo.id where build.id = $1`
		sherpaCheckRow(tx.QueryRow(qrepo, buildID), &repoName, "fetching repo name from database")

		build := _build(tx, repoName, buildID)
		if build.Released != nil {
			userError("Build has been released, cannot be removed")
		}

		_removeBuild(tx, repoName, buildID)
	})
	events <- EventRemoveBuild{repoName, buildID}
}

// CleanupBuilddir cleans up (removes) a build directory.
// This does not remove the build itself from the database.
func (Ding) CleanupBuilddir(ctx context.Context, password, repoName string, buildID int32) (build Build) {
	_checkPassword(password)

	transact(ctx, func(tx *sql.Tx) {
		build = _build(tx, repoName, buildID)
		if build.BuilddirRemoved {
			userError("Builddir already removed")
		}

		err := tx.QueryRow("update build set builddir_removed=true where id=$1 returning id", buildID).Scan(&buildID)
		sherpaCheck(err, "marking builddir as removed in database")

		msg := msg{RemoveBuilddir: &msgRemoveBuilddir{repoName, buildID}}
		err = requestPrivileged(msg)
		sherpaCheck(err, "removing files")

		build = _build(tx, repoName, buildID)
		fillBuild(repoName, &build)
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
	sherpaCheck(err, "listing files in go toolchain dir")

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
			sherpaCheck(err, "reading go symlink for active go toolchain")
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
		sherpaCheck(err, "fetching list of all released go toolchains")
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
		userError("bad goversion")
	}

	switch shortname {
	case "go", "go-prev", "":
	default:
		userError("invalid shortname")
	}

	// Check goversion isn't already installed.
	versionDst := path.Join(config.GoToolchainDir, goversion)
	_, err := os.Stat(versionDst)
	if err == nil {
		userError("already installed")
	}

	releases, err := goreleases.ListAll()
	sherpaCheck(err, "fetching list of all released go toolchains")

	rel := _findRelease(releases, goversion)
	file, err := goreleases.FindFile(rel, runtime.GOOS, runtime.GOARCH, "archive")
	sherpaCheck(err, "finding file for running os and arch")

	msg := msg{InstallGoToolchain: &msgInstallGoToolchain{file, shortname}}
	err = requestPrivileged(msg)
	sherpaCheck(err, "install go toolchain")
}

// RemoveGoToolchain removes a toolchain from go toolchain dir.
// It does not remove a shortname symlink to this toolchain if it exists.
func (Ding) RemoveGoToolchain(ctx context.Context, password, goversion string) {
	_checkPassword(password)

	_checkGoToolchainDir()
	if !validGoversion(goversion) {
		userError("bad goversion")
	}

	msg := msg{RemoveGoToolchain: &msgRemoveGoToolchain{goversion}}
	err := requestPrivileged(msg)
	sherpaCheck(err, "removing go toolchain")
}

// ActivateGoToolchain activates goversion (eg "go1.13.8", "go1.14") under the name
// shortname ("go" or "go-prev"), by creating a symlink in the GoToolchainDir.
func (Ding) ActivateGoToolchain(ctx context.Context, password, goversion, shortname string) {
	_checkPassword(password)

	_checkGoToolchainDir()
	if !validGoversion(goversion) {
		userError("bad goversion")
	}

	switch shortname {
	case "go", "go-prev":
		msg := msg{ActivateGoToolchain: &msgActivateGoToolchain{goversion, shortname}}
		err := requestPrivileged(msg)
		sherpaCheck(err, "removing go toolchain")
	default:
		userError("invalid shortname")
	}
}

func _checkGoToolchainDir() {
	if config.GoToolchainDir == "" {
		userError("GoToolchainDir not configured")
	}
}

func _findRelease(releases []goreleases.Release, goversion string) goreleases.Release {
	for _, rel := range releases {
		if rel.Version == goversion {
			return rel
		}
	}
	userError("version not found")
	return goreleases.Release{}
}
