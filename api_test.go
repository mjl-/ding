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
	tneederr(t, "user:badAuth", func() { api.BuildCancel(ctxbg, "badpass", "repoName", 123) })
	tneederr(t, "user:badAuth", func() { api.BuildCleanupBuilddir(ctxbg, "badpass", "repoName", 123) })
	tneederr(t, "user:badAuth", func() { api.BuildCreate(ctxbg, "badpass", "repoName", "main", "", false) })
	tneederr(t, "user:badAuth", func() { api.Build(ctxbg, "badpass", "repoName", 123) })
	tneederr(t, "user:badAuth", func() { api.BuildRemove(ctxbg, "badpass", 123) })
	tneederr(t, "user:badAuth", func() { api.BuildsCreateLowPrio(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.Builds(ctxbg, "badpass", "repoName") })
	tneederr(t, "user:badAuth", func() { api.ClearRepoHomedirs(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.GoToolchainActivate(ctxbg, "badpass", "go1.23.0", "go") })
	tneederr(t, "user:badAuth", func() { api.GoToolchainInstall(ctxbg, "badpass", "go1.23.0", "go") })
	tneederr(t, "user:badAuth", func() { api.GoToolchainRemove(ctxbg, "badpass", "go1.23.0") })
	tneederr(t, "user:badAuth", func() { api.GoToolchainsListInstalled(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.GoToolchainsListReleased(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.ReleaseCreate(ctxbg, "badpass", "repoName", 123) })
	tneederr(t, "user:badAuth", func() { api.RepoBuilds(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.RepoClearHomedir(ctxbg, "badpass", "repoName") })
	tneederr(t, "user:badAuth", func() { api.RepoCreate(ctxbg, "badpass", Repo{}) })
	tneederr(t, "user:badAuth", func() { api.Repo(ctxbg, "badpass", "repoName") })
	tneederr(t, "user:badAuth", func() { api.RepoRemove(ctxbg, "badpass", "repoName") })
	tneederr(t, "user:badAuth", func() { api.RepoSave(ctxbg, "badpass", Repo{}) })
	tneederr(t, "user:badAuth", func() { api.Settings(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.SettingsSave(ctxbg, "badpass", Settings{}) })
	tneederr(t, "user:badAuth", func() { api.LogLevel(ctxbg, "badpass") })
	tneederr(t, "user:badAuth", func() { api.LogLevelSet(ctxbg, "badpass", LogInfo) })
	tneederr(t, "user:badAuth", func() { api.Version(ctxbg, "badpass") })

	// todo: test more api failures (bad parameters).

	api.Status(ctxbg)

	const buildScript = `#!/usr/bin/env bash
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
	nr := api.RepoCreate(ctxbg, config.Password, r)
	nr.WebhookSecret = ""
	tcompare(t, nr, r)
	r = nr

	// SaveRepo
	r.VCS = VCSCommand
	r.Origin = "sh -c 'echo clone..; mkdir -p checkout/$DING_CHECKOUTPATH; echo commit: ...'"
	nr = api.RepoSave(ctxbg, config.Password, r)

	// Repo
	nr = api.Repo(ctxbg, config.Password, nr.Name)
	tcompare(t, nr.VCS, VCSCommand)
	tcompare(t, nr.Origin, r.Origin)

	// RepoBuilds
	rb := api.RepoBuilds(ctxbg, config.Password)
	tcompare(t, len(rb), 1)
	tcompare(t, rb[0].Repo, nr)

	// RepoClearHomedir
	tneederr(t, "user:error", func() { api.RepoClearHomedir(ctxbg, config.Password, nr.Name) }) // No homedir reuse.

	// ClearRepoHomedirs
	api.ClearRepoHomedirs(ctxbg, config.Password)

	// BuildCreate
	b := api.BuildCreate(ctxbg, config.Password, r.Name, "unused", "", false)

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

	// ReleaseCreate
	rel := api.ReleaseCreate(ctxbg, config.Password, r.Name, b.ID)
	tcompare(t, rel.ID, b.ID)

	// BuildCleanupBuilddir
	api.BuildCleanupBuilddir(ctxbg, config.Password, r.Name, b.ID)

	// Create a build that we will cancel.
	nr = api.Repo(ctxbg, config.Password, nr.Name)
	nr.BuildScript = "#!/usr/bin/env bash\nsleep 3\n"
	nr = api.RepoSave(ctxbg, config.Password, nr)
	cb := api.BuildCreate(ctxbg, config.Password, r.Name, "unused", "", true)
	api.BuildCancel(ctxbg, config.Password, r.Name, cb.ID)
	api.BuildCleanupBuilddir(ctxbg, config.Password, r.Name, cb.ID)
	api.BuildRemove(ctxbg, config.Password, cb.ID)

	api.BuildsCreateLowPrio(ctxbg, config.Password)
	bl = api.Builds(ctxbg, config.Password, r.Name)
	ncancel := 0
	for _, xb := range bl {
		if xb.Finish == nil {
			api.BuildCancel(ctxbg, config.Password, r.Name, xb.ID)
			ncancel++
		}
	}
	tcompare(t, ncancel, 1)
	time.Sleep(100 * time.Millisecond) // todo: get rid of this, waiting for goroutine in CreateLowPrioBuilds to finish before closing the db at end of test.

	// RepoRemove
	api.RepoRemove(ctxbg, config.Password, nr.Name)
	tneederr(t, "user:notFound", func() { api.RepoRemove(ctxbg, config.Password, nr.Name) })

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
	r = api.RepoCreate(ctxbg, config.Password, r)
	b = api.BuildCreate(ctxbg, config.Password, r.Name, r.DefaultBranch, "", false)
	twaitBuild(t, b, StatusSuccess)

	// Test a build with a mercurial repo, if we can run it.
	hgc := exec.Command("hg", "version")
	if _, err := hgc.CombinedOutput(); err == nil {
		hgRepoDir := dingDataDir + "/hgrepo"
		run(workDir, "hg", "init", hgRepoDir)
		run(hgRepoDir, "touch", "file.txt")
		run(hgRepoDir, "hg", "add", "file.txt")
		run(hgRepoDir, "hg", "commit", "-m", "test", "file.txt")

		r = Repo{Name: "hg", VCS: VCSMercurial, Origin: hgRepoDir, DefaultBranch: "default", CheckoutPath: "hgrepo", BuildScript: "#!/usr/bin/env bash\nset -e\necho building...\necho hi>myfile\necho version: 1.2.3\necho release: mycmd linux amd64 toolchain1.2.3 myfile\n"}
		r = api.RepoCreate(ctxbg, config.Password, r)
		b = api.BuildCreate(ctxbg, config.Password, r.Name, r.DefaultBranch, "", false)
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
	api.RepoSave(ctxbg, config.Password, r)

	api.RepoClearHomedir(ctxbg, config.Password, r.Name)
	api.ClearRepoHomedirs(ctxbg, config.Password)

	_, _, settings := api.Settings(ctxbg, config.Password)
	api.SettingsSave(ctxbg, config.Password, settings)

	api.LogLevel(ctxbg, config.Password)
	api.LogLevelSet(ctxbg, config.Password, LogInfo)

	api.Version(ctxbg, config.Password)
}

func TestToolchains(t *testing.T) {
	if os.Getenv("DING_TEST_GOTOOLCHAINS") == "" {
		t.Skip("skipping because DING_TEST_GOTOOLCHAINS is not set")
	}
	t.Log("downloading toolchain because DING_TEST_GOTOOLCHAINS is set")

	testEnv(t)
	api := Ding{}

	installed, active := api.GoToolchainsListInstalled(ctxbg, config.Password)
	tcompare(t, len(installed), 0)
	tcompare(t, active, GoToolchains{})

	released := api.GoToolchainsListReleased(ctxbg, config.Password) // todo: set a timeout
	tcompare(t, len(released) > 0, true)

	api.GoToolchainInstall(ctxbg, config.Password, released[0], "go") // todo: set a timeout

	installed, active = api.GoToolchainsListInstalled(ctxbg, config.Password)
	tcompare(t, installed, released[:1])
	tcompare(t, active, map[string]string{"go": installed[0]})

	api.GoToolchainActivate(ctxbg, config.Password, installed[0], "go-prev")
	_, active = api.GoToolchainsListInstalled(ctxbg, config.Password)
	tcompare(t, active, map[string]string{"go": installed[0], "go-prev": installed[0]})

	api.GoToolchainRemove(ctxbg, config.Password, installed[0])
}
