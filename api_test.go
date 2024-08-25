package main

import (
	"log"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestAPI(t *testing.T) {
	testEnv(t)

	api := Ding{}

	// Check auth for all methods.
	tneederr(t, "user:badAuth", func() { api.ActivateGoToolchain(ctxbg, "badpass", "go1.23.0", "go") })
	tneederr(t, "user:badAuth", func() { api.Build(ctxbg, "badpass", "repoName", 123) })
	tneederr(t, "user:badAuth", func() { api.Builds(ctxbg, "badpass", "repoName") })
	tneederr(t, "user:badAuth", func() { api.CancelBuild(ctxbg, "badpass", "repoName", 123) })
	tneederr(t, "user:badAuth", func() { api.CleanupBuilddir(ctxbg, "badpass", "repoName", 123) })
	tneederr(t, "user:badAuth", func() { api.ClearRepoHomedir(ctxbg, "badpass", "repoName") })
	tneederr(t, "user:badAuth", func() { api.ClearRepoHomedirs(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.CreateBuild(ctxbg, "badpass", "repoName", "main", "") })
	tneederr(t, "user:badAuth", func() { api.CreateBuildLowPrio(ctxbg, "badpass", "repoName", "main", "") })
	tneederr(t, "user:badAuth", func() { api.CreateLowPrioBuilds(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.CreateRelease(ctxbg, "badpass", "repoName", 123) })
	tneederr(t, "user:badAuth", func() { api.CreateRepo(ctxbg, "badpass", Repo{}) })
	tneederr(t, "user:badAuth", func() { api.InstallGoToolchain(ctxbg, "badpass", "go1.23.0", "go") })
	tneederr(t, "user:badAuth", func() { api.ListInstalledGoToolchains(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.ListReleasedGoToolchains(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.Release(ctxbg, "badpass", "repoName", 123) })
	tneederr(t, "user:badAuth", func() { api.RemoveBuild(ctxbg, "badpass", 123) })
	tneederr(t, "user:badAuth", func() { api.RemoveGoToolchain(ctxbg, "badpass", "go1.23.0") })
	tneederr(t, "user:badAuth", func() { api.RemoveRepo(ctxbg, "badpass", "repoName") })
	tneederr(t, "user:badAuth", func() { api.Repo(ctxbg, "badpass", "repoName") })
	tneederr(t, "user:badAuth", func() { api.RepoBuilds(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.SaveRepo(ctxbg, "badpass", Repo{}) })

	// todo: test more api failures (bad parameters).

	api.Status(ctxbg)

	const buildScript = `#!/bin/bash
set -e
echo building...
echo hi>myfile
echo version: 1.2.3
echo release: mycmd linux amd64 toolchain1.2.3 myfile
touch $DING_DOWNLOADDIR/coverage.txt
echo coverage: 80.0
echo coverage-report: coverage.txt
`

	// CreateRepo
	r := Repo{Name: "t0", VCS: VCSGit, Origin: "http://localhost", DefaultBranch: "main", CheckoutPath: "t0", BuildScript: buildScript}
	nr := api.CreateRepo(ctxbg, config.Password, r)
	tcompare(t, nr, r)
	r = nr

	// SaveRepo
	r.VCS = VCSCommand
	r.Origin = "sh -c 'echo clone..; mkdir -p checkout/$DING_CHECKOUTPATH; echo commit: ...'"
	nr = api.SaveRepo(ctxbg, config.Password, r)

	// Repo
	nr = api.Repo(ctxbg, config.Password, nr.Name)
	tcompare(t, nr.VCS, VCSCommand)
	tcompare(t, nr.Origin, r.Origin)

	// RepoBuilds
	rb := api.RepoBuilds(ctxbg, config.Password)
	tcompare(t, len(rb), 1)
	tcompare(t, rb[0].Repo, nr)

	// ClearRepoHomedir
	tneederr(t, "userError", func() { api.ClearRepoHomedir(ctxbg, config.Password, nr.Name) }) // No homedir reuse.

	// ClearRepoHomedirs
	api.ClearRepoHomedirs(ctxbg, config.Password)

	// CreateBuild
	b := api.CreateBuild(ctxbg, config.Password, r.Name, "unused", "")

	// Build
	api.Build(ctxbg, config.Password, r.Name, b.ID)

	// Builds
	bl := api.Builds(ctxbg, config.Password, r.Name)
	tcompare(t, len(bl), 1)
	tcompare(t, bl[0].ID, b.ID)

	// RepoBuilds, now with a build.
	rb = api.RepoBuilds(ctxbg, config.Password)
	tcompare(t, len(rb), 1)
	tcompare(t, len(rb[0].Builds), 1)

	// Wait for build to complete.
	twaitBuild(t, b, StatusSuccess)

	// CreateRelease
	rel := api.CreateRelease(ctxbg, config.Password, r.Name, b.ID)
	tcompare(t, rel.ID, b.ID)

	// Release
	rel = api.Release(ctxbg, config.Password, r.Name, rel.ID)

	// CleanupBuilddir
	api.CleanupBuilddir(ctxbg, config.Password, r.Name, b.ID)

	// Create a build that we will cancel.
	nr = api.Repo(ctxbg, config.Password, nr.Name)
	nr.BuildScript = "#!/bin/bash\nsleep 3\n"
	nr = api.SaveRepo(ctxbg, config.Password, nr)
	cb := api.CreateBuildLowPrio(ctxbg, config.Password, r.Name, "unused", "")
	api.CancelBuild(ctxbg, config.Password, r.Name, cb.ID)
	api.CleanupBuilddir(ctxbg, config.Password, r.Name, cb.ID)
	api.RemoveBuild(ctxbg, config.Password, cb.ID)

	api.CreateLowPrioBuilds(ctxbg, config.Password)
	bl = api.Builds(ctxbg, config.Password, r.Name)
	ncancel := 0
	for _, xb := range bl {
		if xb.Finish == nil {
			api.CancelBuild(ctxbg, config.Password, r.Name, xb.ID)
			ncancel++
		}
	}
	tcompare(t, ncancel, 1)
	time.Sleep(100 * time.Millisecond) // todo: get rid of this, waiting for goroutine in CreateLowPrioBuilds to finish before closing the db at end of test.

	// RepoRemove
	api.RemoveRepo(ctxbg, config.Password, nr.Name)
	tneederr(t, "user:notFound", func() { api.RemoveRepo(ctxbg, config.Password, nr.Name) })

	// Test a build with a git repo. We assume the git binary is present, it is how this repo was cloned.
	workDir, err := os.Getwd()
	tcheck(t, err, "get workdir")
	run := func(dir, cmd string, args ...string) {
		t.Helper()
		c := exec.Command(cmd, args...)
		c.Dir = dir
		output, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("run in %q, command %q, args %v: %v, output: %s", dir, cmd, args, err, output)
		}
	}
	gitRepoDir := dingDataDir + "/gitrepo"
	run(workDir, "git", "init", "--initial-branch=main", gitRepoDir)
	run(gitRepoDir, "touch", "file.txt")
	run(gitRepoDir, "git", "add", "file.txt")
	run(gitRepoDir, "git", "commit", "-m", "test", "file.txt")

	r = Repo{Name: "g0", VCS: VCSGit, Origin: gitRepoDir, DefaultBranch: "main", CheckoutPath: "g0", BuildScript: buildScript}
	r = api.CreateRepo(ctxbg, config.Password, r)
	b = api.CreateBuild(ctxbg, config.Password, r.Name, r.DefaultBranch, "")
	twaitBuild(t, b, StatusSuccess)

	// Test a build with a mercurial repo, if we can run it.
	hgc := exec.Command("hg", "version")
	if _, err := hgc.CombinedOutput(); err == nil {
		hgRepoDir := dingDataDir + "/hgrepo"
		run(workDir, "hg", "init", hgRepoDir)
		run(hgRepoDir, "touch", "file.txt")
		run(hgRepoDir, "hg", "add", "file.txt")
		run(hgRepoDir, "hg", "commit", "-m", "test", "file.txt")

		r = Repo{Name: "hg", VCS: VCSMercurial, Origin: hgRepoDir, DefaultBranch: "default", CheckoutPath: "hgrepo", BuildScript: "#!/bin/bash\nset -e\necho building...\necho hi>myfile\necho version: 1.2.3\necho release: mycmd linux amd64 toolchain1.2.3 myfile\n"}
		r = api.CreateRepo(ctxbg, config.Password, r)
		b = api.CreateBuild(ctxbg, config.Password, r.Name, r.DefaultBranch, "")
		twaitBuild(t, b, StatusSuccess)
	} else {
		log.Printf("not testing build with mercurial repository")
	}

	// Tests that require running as root.
	if os.Getuid() != 0 {
		return
	}

	// Use git repo g0.
	r = api.Repo(ctxbg, config.Password, "g0")
	// Enable using the same homedir for builds.
	var one uint32 = 1
	r.UID = &one
	api.SaveRepo(ctxbg, config.Password, r)

	api.ClearRepoHomedir(ctxbg, config.Password, r.Name)
	api.ClearRepoHomedirs(ctxbg, config.Password)
}

func TestToolchains(t *testing.T) {
	if os.Getenv("DING_TEST_GOTOOLCHAINS") == "" {
		t.Skip("skipping because DING_TEST_GOTOOLCHAINS is not set")
	}
	t.Log("downloading toolchain because DING_TEST_GOTOOLCHAINS is set")

	testEnv(t)
	api := Ding{}

	installed, active := api.ListInstalledGoToolchains(ctxbg, config.Password)
	tcompare(t, len(installed), 0)
	tcompare(t, len(active), 0)

	released := api.ListReleasedGoToolchains(ctxbg, config.Password) // todo: set a timeout
	tcompare(t, len(released) > 0, true)

	api.InstallGoToolchain(ctxbg, config.Password, released[0], "go") // todo: set a timeout

	installed, active = api.ListInstalledGoToolchains(ctxbg, config.Password)
	tcompare(t, installed, released[:1])
	tcompare(t, active, map[string]string{"go": installed[0]})

	api.ActivateGoToolchain(ctxbg, config.Password, installed[0], "go-prev")
	_, active = api.ListInstalledGoToolchains(ctxbg, config.Password)
	tcompare(t, active, map[string]string{"go": installed[0], "go-prev": installed[0]})

	api.RemoveGoToolchain(ctxbg, config.Password, installed[0])
}
