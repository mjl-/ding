package main

import (
	"time"
)

// Settings holds runtime configuration options.
type Settings struct {
	ID                     int32    // singleton with ID 1
	NotifyEmailAddrs       []string // Email address to notify on build breakage/fixage. Can be overridden per repository.
	GithubWebhookSecret    string   // Secret for webhooks from github. Migrated from config. New repo's get their own unique secret on creation.
	GiteaWebhookSecret     string
	BitbucketWebhookSecret string
	RunPrefix              []string // Commands prefixed to the clone and build commands. E.g. /usr/bin/nice.
	Environment            []string // Additional environment variables to set during clone and build.

	// If set, new "go", "go-prev" and "go-next" (if present, for release candidates)
	// are automatically downloaded and installed (symlinked as active).
	AutomaticGoToolchains bool
}

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
	Name                      string  // Short name for repo, typically last element of repo URL/path.
	VCS                       VCS     `bstore:"nonzero"`
	Origin                    string  `bstore:"nonzero"` // git/mercurial "URL" (as understood by the respective commands), often SSH or HTTPS. if `vcs` is `command`, this is executed using sh.
	DefaultBranch             string  // Name of default branch, e.g. "main" or "master" for git, or "default" for mercurial, empty for command.
	CheckoutPath              string  `bstore:"nonzero"` // Path to place the checkout in.
	BuildScript               string  // Shell scripts that compiles the software, runs tests, and creates releasable files.
	UID                       *uint32 // If set, fixed uid to use for builds, sharing a home directory where files can be cached, to speed up builds.
	HomeDiskUsage             int64   // Disk usage of shared home directory after last finished build. Only if UID is set.
	WebhookSecret             string  // If non-empty, a per-repo secret for incoming webhook calls.
	AllowGlobalWebhookSecrets bool    // If set, global webhook secrets are allowed to start builds. Set initially during migrations. Will be ineffective when global webhooks have been unconfigured.

	// If true, build is run with bubblewrap (bwrap) to isolate the environment
	// further. Only the system, the build directory, home directory and toolchain
	// directory is available.
	Bubblewrap bool
	// If true, along with Bubblewrap, then no network access is possible during the
	// build (though it is during clone).
	BubblewrapNoNet bool

	// If not empty, each address gets notified about build
	// breakage/fixage, overriding the default address configured in the
	// configuration file.
	NotifyEmailAddrs []string

	// If set, automatically installed Go toolchains will trigger a low priority build
	// for this repository.
	BuildOnUpdatedToolchain bool
}

// Build is an attempt at building a repository.
type Build struct {
	ID                 int32
	RepoName           string      `bstore:"nonzero,ref Repo"`
	Branch             string      `bstore:"nonzero,index"`
	CommitHash         string      // Can be empty until `checkout` step, when building latest version of a branch.
	Status             BuildStatus `bstore:"nonzero"`
	Created            time.Time   `bstore:"default now"` // Time of creation of this build. Ding only has one concurrent build per repo, so the start time may be later.
	Start              *time.Time  // Time the build was started. Duration of a build is finish - start.
	Finish             *time.Time
	ErrorMessage       string
	Released           *time.Time // Once set, this build itself won't be removed from the database, but its build directory may be removed.
	BuilddirRemoved    bool
	Coverage           *float32 // Test coverage in percentage, from 0 to 100.
	CoverageReportFile string   // Relative to URL /dl/<reponame>/<buildid>.
	Version            string   // Version if this build, typically contains a semver version, with optional commit count/hash, perhaps a branch.
	BuildScript        string

	// Low-prio builds run after regular builds for a repo have finished. And we only
	// run one low-prio build in ding at a time. Useful after a toolchain update.
	LowPrio bool

	LastLine  string // Last line of output, when build has completed.
	DiskUsage int64  // Disk usage for build.

	// Change in disk usage of shared home directory, if enabled for this repository.
	// Disk usage can shrink, e.g. after a cleanup.
	HomeDiskUsageDelta int64

	Results []Result // Only set for success builds.

	Steps []Step // Only set for finished builds.
}

// Result is a file created during a build, as the result of a build.
type Result struct {
	Command   string // Short name of command, without version, as you would want to run it from a command-line.
	Os        string // eg `any`, `linux`, `darwin, `openbsd`, `windows`.
	Arch      string // eg `any`, `amd64`, `arm64`.
	Toolchain string // String describing the tools used during build, eg SDK version.

	// Path relative to the checkout directory where build.sh is run.
	// For builds, the file is started at <dataDir>/build/<repoName>/<buildID>/checkout/<checkoutPath>/<filename>.
	// For releases, the file is stored gzipped at <dataDir>/release/<repoName>/<buildID>/<basename of filename>.gz.
	Filename string
	Filesize int64 // Size of filename.
}

// Step is one phase of a build and stores the output generated in that step.
type Step struct {
	Name   string // Mostly same values as build.status.
	Output string // Combined output of stdout and stderr.
	Nsec   int64  // Time it took this step to finish, initially 0.
}
