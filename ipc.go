package main

import (
	"log/slog"
	"os"

	"github.com/mjl-/goreleases"
)

// Start a build, running build.sh.
type msgBuild struct {
	RepoName        string
	BuildID         int32
	UID             uint32 // UID to run this build under. Ignored if IsolateBuilds is entirely off. Otherwise it is set to either a unique UID, or a fixed UID per repo, depending on configuration.
	CheckoutPath    string
	RunPrefix       []string // From settings.
	Env             []string // Including from settings.Environment
	ToolchainDir    string
	HomeDir         string
	Bubblewrap      bool
	BubblewrapNoNet bool

	// If non-empty, we do builds for one or more go toolchains.
	GoToolchains GoToolchains

	// Whether the reason for this build was the installation of a new Go toolchain.
	NewGoToolchain bool
}

// Chown the home, checkout and download dir of a build.
// Called before starting a build.
type msgChown struct {
	RepoName   string
	BuildID    int32
	SharedHome bool
	UID        uint32
}

// Remove a build directory. Called for automatic cleanup or by explicit user request.
type msgRemoveBuilddir struct {
	RepoName string
	BuildID  int32
}

// Remove all files for a repo, including all builds, releases, possibly shared home directory.
type msgRemoveRepo struct {
	RepoName string
}

// Remove shared home directory for a repository. Called by explicit user request.
type msgRemoveSharedHome struct {
	RepoName string
}

// Cancel potentially running command by buildID.
type msgCancelCommand struct {
	BuildID int32
}

// Install released go toolchain into GoToolchainDir.
type msgInstallGoToolchain struct {
	File      goreleases.File
	Shortname string // "go", "goprev" or "gonext"
}

// Remove installed go toolchain from GoToolchainDir, leaving shortname symlink untouched.
type msgRemoveGoToolchain struct {
	Goversion string // eg "go1.14" or "go1.13.8"
}

// Activate installed go toolchain in GoToolchainDir under shortname by creating a symlink.
type msgActivateGoToolchain struct {
	Goversion string // eg "go1.14 or "go1.13.8"
	Shortname string // "go", "goprev" or "gonext"
}

// Lookup latest Go toolchains and update to them, setting goprev, go, and possibly gonext symlinks.
type msgAutomaticGoToolchain struct {
}

type msgLogLevelSet struct {
	LogLevel slog.Level
}

// Message from unprivileged webserver to root process.
// Only the first non-nil field is handled.
type msg struct {
	Build                *msgBuild
	Chown                *msgChown
	RemoveBuilddir       *msgRemoveBuilddir
	RemoveRepo           *msgRemoveRepo
	RemoveSharedHome     *msgRemoveSharedHome
	CancelCommand        *msgCancelCommand
	InstallGoToolchain   *msgInstallGoToolchain
	RemoveGoToolchain    *msgRemoveGoToolchain
	ActivateGoToolchain  *msgActivateGoToolchain
	AutomaticGoToolchain *msgAutomaticGoToolchain
	LogLevelSet          *msgLogLevelSet
}

// request from one of the http handlers to httpserve's request mux
type request struct {
	msg           msg
	errorResponse chan error
	buildResponse chan buildResult
}

// result of starting a build
type buildResult struct {
	err    error // If non-nil, quick failure. Otherwise, the files below will send updates.
	stdout *os.File
	stderr *os.File
	status *os.File // We read a gob-encoded string from status as the exit string.
}

func requestPrivileged(msg msg) error {
	req := request{
		msg,
		make(chan error),
		nil,
	}
	rootRequests <- req
	return <-req.errorResponse
}
