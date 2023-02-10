package main

import (
	"time"
)

// Repo is a repository as stored in the database.
type Repo struct {
	ID            int32   `json:"id"`
	Name          string  `json:"name"`            // short name for repo, typically last element of repo URL/path
	VCS           string  `json:"vcs"`             // `git`, `mercurial` or `command`
	Origin        string  `json:"origin"`          // git/mercurial "URL" (as understood by the respective commands), often SSH or HTTPS. if `vcs` is `command`, this is executed using sh.
	DefaultBranch string  `json:"default_branch"`  // Name of default branch, e.g. "main" or "master" for git, or "default" for mercurial.
	CheckoutPath  string  `json:"checkout_path"`   // path to place the checkout in.
	BuildScript   string  `json:"build_script"`    // shell scripts that compiles the software, runs tests, and creates releasable files.
	UID           *uint32 `json:"uid"`             // If set, fixed uid to use for builds, sharing a home directory where files can be cached, to speed up builds.
	HomeDiskUsage int64   `json:"home_disk_usage"` // Disk usage of shared home directory after last finished build. Only if UID is set.
}

// RepoBuilds is a repository and its recent builds, per branch.
type RepoBuilds struct {
	Repo   Repo    `json:"repo"`
	Builds []Build `json:"builds"`
}

// Result is a file created during a build, as the result of a build. Files like this can be released.
type Result struct {
	Command   string `json:"command"`   // short name of command, without version, as you would want to run it from a command-line
	Os        string `json:"os"`        // eg `any`, `linux`, `darwin, `openbsd`, `windows`
	Arch      string `json:"arch"`      // eg `any`, `amd64`, `arm64`
	Toolchain string `json:"toolchain"` // string describing the tools used during build, eg SDK version
	Filename  string `json:"filename"`  // path relative to the checkout directory where build.sh is run
	Filesize  int64  `json:"filesize"`  // size of filename
}

// Build is an attempt at building a repository.
type Build struct {
	ID                 int32      `json:"id"`
	RepoID             int32      `json:"repo_id"`
	Branch             string     `json:"branch"`
	CommitHash         string     `json:"commit_hash"` // can be empty until `checkout` step, when building latest version of a branch
	Status             string     `json:"status"`      // `new`, `clone`, `checkout`, `build`, `success`
	Created            time.Time  `json:"created"`     // Time of creation of this build. Ding only has one concurrent build per repo, so the start time may be later.
	Start              *time.Time `json:"start"`       // Time the build was started. Of a build is finish - start.
	Finish             *time.Time `json:"finish"`
	ErrorMessage       string     `json:"error_message"`
	Released           *time.Time `json:"released"`
	BuilddirRemoved    bool       `json:"builddir_removed"`
	Coverage           *float32   `json:"coverage"`             // Test coverage in percentage, from 0 to 100.
	CoverageReportFile string     `json:"coverage_report_file"` // Relative to URL /dl/<reponame>/<buildid>.
	Version            string     `json:"version"`              // Version if this build, typically contains a semver version, with optional commit count/hash, perhaps a branch.

	// Low-prio builds run after regular builds for a repo have finished. And we only
	// run one low-prio build in ding at a time. Useful after a toolchain update.
	LowPrio bool `json:"low_prio"`

	LastLine  string `json:"last_line"`  // Last line from last steps output.
	DiskUsage int64  `json:"disk_usage"` // Disk usage for build.

	// Change in disk usage of shared home directory, if enabled for this repository.
	// Disk usage can shrink, e.g. after a cleanup.
	HomeDiskUsageDelta int64 `json:"home_disk_usage_delta"`

	Results []Result `json:"results"`
}

// Step is one phase of a build and stores the output generated in that step.
type Step struct {
	Name   string `json:"name"` // same values as build.status
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Output string `json:"output"` // combined output of stdout and stderr
	Nsec   int64  `json:"nsec"`   // time it took this step to finish, initially 0
}

// BuildResult is the stored result of a build, including the build script and step outputs.
type BuildResult struct {
	Build       Build  `json:"build"`
	BuildScript string `json:"build_script"`
	Steps       []Step `json:"steps"`
}
