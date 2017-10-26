package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/mjl/sherpa"
)

var (
	stepNames = []string{
		"clone",
		"checkout",
		"build",
		"test",
		"release",
		"success",
	}
)

// The Ding API lets you compile git branches, build binaries, run tests, and publish binaries.
type Ding struct {
}

// Checks program status.
// If backend connectivity is broken, this sherpa call results in a 500 internal server error. Useful for monitoring tools.
func (Ding) Status() {
}

// Build a specific commit in the background, returning immediately.
func (Ding) BuildStart(repoName, branch, commit string) {
	defer func() {
		if err := recover(); err != nil {
			if serr, ok := err.(*sherpa.Error); ok {
				if serr.Code != "userError" {
					log.Println("background build failed:", serr.Message)
				}
			}
		}
	}()
	go Ding{}.Build(repoName, branch, commit)
}

// Build a specific commit.
func (Ding) Build(repoName, branch, commit string) (build Build) {
	var repo Repo
	var buildDir string
	workdir, err := os.Getwd()
	sherpaCheck(err, "getting current work dir")
	transact(func(tx *sql.Tx) {
		repo = _repo(tx, repoName)

		q := `insert into build (repo_id, branch, commit_hash, status, start) values ($1, $2, $3, $4, NOW()) returning id`
		checkParseRow(tx.QueryRow(q, repo.Id, branch, commit, "new"), &build.Id, "inserting new build into database")

		buildDir = fmt.Sprintf("%s/build/%s/%d", workdir, repo.Name, build.Id)
		err := os.MkdirAll(buildDir, 0777)
		sherpaCheck(err, "creating build dir")
	})

	err = os.MkdirAll(buildDir+"/scripts", 0777)
	sherpaCheck(err, "creating scripts dir")
	err = os.MkdirAll(buildDir+"/home", 0777)
	sherpaCheck(err, "creating home dir")

	scriptsDir := buildDir + "/scripts/"
	for _, script := range []string{"build.sh", "test.sh", "release.sh"} {
		_copyScript(fmt.Sprintf("config/%s/%s", repo.Name, script), scriptsDir+script)
	}

	_updateStatus := func(status string) {
		_, err := database.Exec("update build set status=$1 where id=$2", status, build.Id)
		sherpaCheck(err, "updating build status in database")
	}

	outputDir := buildDir + "/output"
	err = os.MkdirAll(outputDir, 0777)
	sherpaCheck(err, "creating output dir")

	_updateStatus("clone")
	err = run(build.Id, "clone", buildDir, buildDir, "git", "clone", "--branch", branch, repo.Origin, "checkout/"+repo.Name)
	sherpaUserCheck(err, "cloning repository")
	checkoutDir := fmt.Sprintf("%s/checkout/%s", buildDir, repo.Name)

	_updateStatus("checkout")
	err = run(build.Id, "checkout", buildDir, checkoutDir, "git", "checkout", commit)
	sherpaUserCheck(err, "checkout out revision")

	_updateStatus("build")
	err = run(build.Id, "build", buildDir, checkoutDir, "../../scripts/build.sh")
	sherpaUserCheck(err, "building")

	_updateStatus("test")
	err = run(build.Id, "test", buildDir, checkoutDir, "../../scripts/test.sh")
	sherpaUserCheck(err, "running tests")

	_updateStatus("release")
	err = run(build.Id, "release", buildDir, checkoutDir, "../../scripts/release.sh")
	sherpaUserCheck(err, "publishing build")

	transact(func(tx *sql.Tx) {
		results := parseResults(outputDir + "/release.stdout")

		qins := `insert into result (build_id, command, version, os, arch, toolchain, filename) values ($1, $2, $3, $4, $5, $6, $7) returning id`
		for _, result := range results {
			if !strings.HasPrefix(result.Filename, "/") {
				result.Filename = checkoutDir + "/" + result.Filename
			}
			_, err := os.Stat(result.Filename)
			sherpaUserCheck(err, "testing whether released file exists")
			var id int
			err = tx.QueryRow(qins, build.Id, result.Command, result.Version, result.Os, result.Arch, result.Toolchain, path.Base(result.Filename)).Scan(&id)
			sherpaCheck(err, "inserting result into database")
			fileCopy(result.Filename, fmt.Sprintf("release/%s/%s", repo.Name, path.Base(result.Filename)))
		}

		_, err = tx.Exec("update build set status='success', finish=NOW() where id=$1", build.Id)
		sherpaCheck(err, "marking build as success in database")

		build = _build(tx, build.Id)
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
	defer func() {
		err2 := df.Close()
		if err == nil {
			err = err2
		}
		if err != nil {
			os.Remove(dst)
			sherpaCheck(err, "installing result file")
		}
	}()
	_, err = io.Copy(df, sf)
	sherpaCheck(err, "copying result file to destination")
}

func parseResults(path string) (results []Result) {
	f, err := os.Open(path)
	sherpaUserCheck(err, "opening release output")
	defer func() {
		sherpaUserCheck(f.Close(), "closing release output")
	}()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// lines should be of the form:
		//  "release:" command version os arch toolchain path
		line := scanner.Text()
		t := strings.Split(line, " ")
		if t[0] != "release:" {
			continue
		}
		if len(t) != 7 {
			sherpaUserCheck(err, "invalid output line, should have 7 words: "+line)
		}
		results = append(results, Result{t[1], t[2], t[3], t[4], t[5], t[6]})
	}
	err = scanner.Err()
	sherpaUserCheck(err, "reading release output")
	return
}

func _copyScript(src, dst string) {
	in, err := os.Open(src)
	sherpaCheck(err, "copy script: opening source script")
	out, err := os.Create(dst)
	sherpaCheck(err, "copy script: opening destinations script")
	_, err = io.Copy(out, in)
	sherpaCheck(err, "copy script: copying data")
	err = in.Close()
	sherpaCheck(err, "copy script: closing input")
	err = out.Close()
	sherpaCheck(err, "copy script: closing output")
	err = os.Chmod(dst, os.FileMode(0755))
	sherpaCheck(err, "copy script: chmod")
}

func run(buildId int, stage, buildDir, workDir, command string, args ...string) (err error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workDir
	cmd.Env = []string{
		fmt.Sprintf("HOME=%s/home", buildDir),
		fmt.Sprintf("BUILDID=%d", buildId),
	}
	var output, stdout, stderr, nsecFile io.WriteCloser
	t0 := time.Now()
	defer func() {
		check := func(err2 error) {
			if err == nil {
				err = err2
			}
		}
		if output != nil {
			check(output.Close())
		}
		if stdout != nil {
			check(stdout.Close())
		}
		if stderr != nil {
			check(stderr.Close())
		}

		if nsecFile != nil {
			_, err2 := fmt.Fprintf(nsecFile, "%d", time.Now().Sub(t0))
			check(err2)
		}
	}()
	if output, err = os.Create(buildDir + "/output/" + stage + ".output"); err != nil {
		return fmt.Errorf("creating combined output file: %s", err)
	}
	output = LineWriter(output)
	if stdout, err = os.Create(buildDir + "/output/" + stage + ".stdout"); err != nil {
		return fmt.Errorf("creating stdout file: %s", err)
	}
	stdout = LineWriter(stdout)
	cmd.Stdout = io.MultiWriter(stdout, output)

	if stderr, err = os.Create(buildDir + "/output/" + stage + ".stderr"); err != nil {
		return fmt.Errorf("opening stderr file: %s", err)
	}
	stderr = LineWriter(stderr)
	cmd.Stderr = io.MultiWriter(stderr, output)

	if nsecFile, err = os.Create(buildDir + "/output/" + stage + ".nsec"); err != nil {
		return fmt.Errorf("opening nsec file: %s", err)
	}

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("running command: %s", err)
	}
	if err = output.Close(); err != nil {
		return err
	}
	output = nil
	if err = stdout.Close(); err != nil {
		return err
	}
	stdout = nil
	if err = stderr.Close(); err != nil {
		return err
	}
	stderr = nil
	return nil
}

// RepoBuilds returns all repositories and their latest build per branch (always for master & develop, for other branches only if the latest build was less than 4 weeks ago).
func (Ding) RepoBuilds() (rb []RepoBuilds) {
	q := `
		with repo_branch_builds as (
			select *
			from build_with_result
			where id in (
				select max(id) as id
				from build
				where branch in ('master', 'develop') or start > now() - interval '4 weeks'
				group by repo_id, branch
			)
		)
		select json_agg(repobuilds.*)
		from (
			select row_to_json(repo.*) as repo, array_remove(array_agg(rbb.*), null) as builds
			from repo
			left join repo_branch_builds rbb on repo.id = rbb.repo_id
			group by repo.id
		) repobuilds
	`
	checkParseRow(database.QueryRow(q), &rb, "fetching repobuilds")
	return
}

func (Ding) Repo(repoName string) (repo Repo) {
	transact(func(tx *sql.Tx) {
		repo = _repo(tx, repoName)
	})
	return
}

// Builds returns builds for a repo
func (Ding) Builds(repoName string) (builds []Build) {
	q := `select coalesce(json_agg(bwr.* order by start desc), '[]') from build_with_result bwr join repo on bwr.repo_id = repo.id where repo.name=$1`
	checkParseRow(database.QueryRow(q, repoName), &builds, "fetching builds")
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

// Create new repository.
func (Ding) CreateRepo(repo Repo) (r Repo) {
	transact(func(tx *sql.Tx) {
		q := `insert into repo (name, origin) values ($1, $2) returning id`
		var id int64
		checkParseRow(tx.QueryRow(q, repo.Name, repo.Origin), &id, "inserting repository in database")
		r = _repo(tx, repo.Name)

		configDir := fmt.Sprintf("config/%s/", repo.Name)
		err := os.MkdirAll(configDir, 0777)
		sherpaCheck(err, "creating config dir for new repository")
		writeFile(configDir+"build.sh", "")
		writeFile(configDir+"test.sh", "")
		writeFile(configDir+"release.sh", "")
	})
	return
}

// Save repository.
func (Ding) SaveRepo(repo Repo, repoConfig RepoConfig) (r Repo, rc RepoConfig) {
	transact(func(tx *sql.Tx) {
		q := `update repo set name=$1, origin=$2 where id=$3 returning id`
		checkParseRow(tx.QueryRow(q, repo.Name, repo.Origin, repo.Id), &repo.Id, "updating repo in database")

		configDir := fmt.Sprintf("config/%s/", repo.Name)
		writeFile(configDir+"build.sh", repoConfig.BuildScript)
		writeFile(configDir+"test.sh", repoConfig.TestScript)
		writeFile(configDir+"release.sh", repoConfig.ReleaseScript)

		r = _repo(tx, repo.Name)
		rc = _repoConfig(repo.Name)
	})
	return
}

// Remove repository and all its builds.
func (Ding) RemoveRepo(repoName string) {
	transact(func(tx *sql.Tx) {
		_, err := tx.Exec(`delete from result where build_id in (select id from build where repo_id in (select id from repo where name=$1))`, repoName)
		sherpaCheck(err, "removing results from database")

		_, err = tx.Exec(`delete from build where repo_id in (select id from repo where name=$1)`, repoName)
		sherpaCheck(err, "removing builds from database")

		var id int
		checkParseRow(tx.QueryRow(`delete from repo where name=$1 returning id`, repoName), &id, "removing repo from database")
	})
	err := os.RemoveAll(fmt.Sprintf("config/%s", repoName))
	sherpaCheck(err, "removing config directory")

	err = os.RemoveAll(fmt.Sprintf("build/%s", repoName))
	sherpaCheck(err, "removing build directory")

	err = os.RemoveAll(fmt.Sprintf("release/%s", repoName))
	sherpaCheck(err, "removing release directory")
}

func readFile(path string) string {
	f, err := os.Open(path)
	sherpaCheck(err, "opening script")
	buf, err := ioutil.ReadAll(f)
	err2 := f.Close()
	if err == nil {
		err = err2
	}
	sherpaCheck(err, "reading script")
	return string(buf)
}

func parseInt(s string) int64 {
	v, err := strconv.ParseInt(s, 10, 64)
	sherpaCheck(err, "parsing integer")
	return v
}

func _repoConfig(repoName string) (rc RepoConfig) {
	rc.BuildScript = readFile(fmt.Sprintf("config/%s/build.sh", repoName))
	rc.TestScript = readFile(fmt.Sprintf("config/%s/test.sh", repoName))
	rc.ReleaseScript = readFile(fmt.Sprintf("config/%s/release.sh", repoName))
	return
}

func (Ding) RepoConfig(repoName string) (rc RepoConfig) {
	transact(func(tx *sql.Tx) {
		_repo(tx, repoName)
	})
	rc = _repoConfig(repoName)
	return
}

func (Ding) BuildResult(repoName string, buildId int) (br BuildResult) {
	transact(func(tx *sql.Tx) {
		br.Build = _build(tx, buildId)
	})

	buildDir := fmt.Sprintf("build/%s/%d/", repoName, buildId)
	br.RepoConfig.BuildScript = readFile(buildDir + "scripts/build.sh")
	br.RepoConfig.TestScript = readFile(buildDir + "scripts/test.sh")
	br.RepoConfig.ReleaseScript = readFile(buildDir + "scripts/release.sh")

	outputDir := buildDir + "output/"
	for _, stepName := range stepNames {
		var step Step
		if stepName == "success" {
			step.Name = "success"
		} else {
			step = Step{
				Name:   stepName,
				Stdout: readFile(outputDir + stepName + ".stdout"),
				Stderr: readFile(outputDir + stepName + ".stderr"),
				Output: readFile(outputDir + stepName + ".output"),
				Nsec:   parseInt(readFile(outputDir + stepName + ".nsec")),
			}
		}
		br.Steps = append(br.Steps, step)
		if stepName == br.Build.Status {
			break
		}
	}
	return
}

// Remove build. Both from database and all local files.
func (Ding) RemoveBuild(buildId int) {
	var repoName string
	transact(func(tx *sql.Tx) {
		qrepo := `select to_json(repo.name) from build join repo on build.repo_id = repo.id where build.id = $1`
		checkParseRow(tx.QueryRow(qrepo, buildId), &repoName, "fetching repo name from database")

		var filenames []string
		qres := `select coalesce(json_agg(filename), '[]') from result where build_id=$1`
		checkParseRow(tx.QueryRow(qres, buildId), &filenames, "fetching released files")

		_, err := tx.Exec(`delete from result where build_id=$1`, buildId)
		sherpaCheck(err, "removing results from database")

		q := `delete from build where id=$1 returning id`
		checkParseRow(tx.QueryRow(q, buildId), &buildId, "removing build from database")

		for _, filename := range filenames {
			path := fmt.Sprintf("release/%s/%s", repoName, filename)
			err = os.Remove(path)
			if err != nil {
				log.Printf("removing %s: %s\n", path, err)
			}
		}
	})
	buildDir := fmt.Sprintf("build/%s/%d", repoName, buildId)
	err := os.RemoveAll(buildDir)
	sherpaCheck(err, "removing build directory")
}