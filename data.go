package main

import (
	"time"
)

// BuildStatus indicates the progress of a build.
type BuildStatus string

// Statuses a build goes through. If a build fails, it will have a non-nil Finish,
// and the status indicates the step that failed.
const (
	StatusNew       BuildStatus = "new"       // Build queued but not yet started.
	StatusClone     BuildStatus = "clone"     // Cloning source code, e.g. from git.
	StatusBuild     BuildStatus = "build"     // Building application.
	StatusSuccess   BuildStatus = "success"   // Build was successful.
	StatusCancelled BuildStatus = "cancelled" // Build was cancelled before finishing.
)

// VCS indicates the mechanism to fetch the source code.
type VCS string

// Version control systems, for cloning code of a repository.
const (
	VCSGit       VCS = "git"
	VCSMercurial VCS = "mercurial"
	// Custom shell script that will do the cloning. Escape hatch mechanism to support
	// past/future systems.
	VCSCommand VCS = "command"
)

// Repo is a repository as stored in the database.
type Repo struct {
	Name          string  `json:"name"` // short name for repo, typically last element of repo URL/path
	VCS           VCS     `bstore:"nonzero" json:"vcs"`
	Origin        string  `bstore:"nonzero" json:"origin"`        // git/mercurial "URL" (as understood by the respective commands), often SSH or HTTPS. if `vcs` is `command`, this is executed using sh.
	DefaultBranch string  `json:"default_branch"`                 // Name of default branch, e.g. "main" or "master" for git, or "default" for mercurial.
	CheckoutPath  string  `bstore:"nonzero" json:"checkout_path"` // path to place the checkout in.
	BuildScript   string  `json:"build_script"`                   // shell scripts that compiles the software, runs tests, and creates releasable files.
	UID           *uint32 `json:"uid"`                            // If set, fixed uid to use for builds, sharing a home directory where files can be cached, to speed up builds.
	HomeDiskUsage int64   `json:"home_disk_usage"`                // Disk usage of shared home directory after last finished build. Only if UID is set.
}

// Build is an attempt at building a repository.
type Build struct {
	ID                 int32       `json:"id"`
	RepoName           string      `bstore:"nonzero,ref Repo" json:"repo_name"`
	Branch             string      `bstore:"nonzero,index" json:"branch"`
	CommitHash         string      `json:"commit_hash"` // can be empty until `checkout` step, when building latest version of a branch
	Status             BuildStatus `bstore:"nonzero" json:"status"`
	Created            time.Time   `bstore:"default now" json:"created"` // Time of creation of this build. Ding only has one concurrent build per repo, so the start time may be later.
	Start              *time.Time  `json:"start"`                        // Time the build was started. Of a build is finish - start.
	Finish             *time.Time  `json:"finish"`
	ErrorMessage       string      `json:"error_message"`
	Released           *time.Time  `json:"released"` // Once set, this build itself won't be removed from the database, but its build directory may be removed.
	BuilddirRemoved    bool        `json:"builddir_removed"`
	Coverage           *float32    `json:"coverage"`             // Test coverage in percentage, from 0 to 100.
	CoverageReportFile string      `json:"coverage_report_file"` // Relative to URL /dl/<reponame>/<buildid>.
	Version            string      `json:"version"`              // Version if this build, typically contains a semver version, with optional commit count/hash, perhaps a branch.
	BuildScript        string      `json:"build_script"`

	// Low-prio builds run after regular builds for a repo have finished. And we only
	// run one low-prio build in ding at a time. Useful after a toolchain update.
	LowPrio bool `json:"low_prio"`

	LastLine  string `json:"last_line"`  // Last line of output, when build has completed.
	DiskUsage int64  `json:"disk_usage"` // Disk usage for build.

	// Change in disk usage of shared home directory, if enabled for this repository.
	// Disk usage can shrink, e.g. after a cleanup.
	HomeDiskUsageDelta int64 `json:"home_disk_usage_delta"`

	Results []Result `json:"results"` // Only set for success builds.

	Steps []Step `json:"steps"` // Only set for finished builds.
}

// Result is a file created during a build, as the result of a build.
type Result struct {
	Command   string `json:"command"`   // short name of command, without version, as you would want to run it from a command-line
	Os        string `json:"os"`        // eg `any`, `linux`, `darwin, `openbsd`, `windows`
	Arch      string `json:"arch"`      // eg `any`, `amd64`, `arm64`
	Toolchain string `json:"toolchain"` // string describing the tools used during build, eg SDK version

	// Path relative to the checkout directory where build.sh is run.
	// For builds, the file is started at <dataDir>/build/<repoName>/<buildID>/checkout/<checkoutPath>/<filename>.
	// For releases, the file is stored gzipped at <dataDir>/release/<repoName>/<buildID>/<basename of filename>.gz.
	Filename string `json:"filename"`
	Filesize int64  `json:"filesize"` // size of filename
}

// Step is one phase of a build and stores the output generated in that step.
type Step struct {
	Name   BuildStatus `json:"name"`   // same values as build.status
	Output string      `json:"output"` // combined output of stdout and stderr
	Nsec   int64       `json:"nsec"`   // time it took this step to finish, initially 0
}
