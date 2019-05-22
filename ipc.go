package main

import (
	"os"
)

// Start a build, running build.sh.
type msgBuild struct {
	RepoName     string
	BuildID      int
	UID          int32 // UID to run this build under. Ignored if IsolateBuilds is entirely off. Otherwise it is set to either a unique UID, or a fixed UID per repo, depending on configuration.
	CheckoutPath string
	Env          []string
}

// Chown the home, checkout and download dir of a build.
// Called before starting a build.
type msgChown struct {
	RepoName   string
	BuildID    int
	SharedHome bool
	UID        int32
}

// Remove a build directory. Called for automatic cleanup or by explicit user request.
type msgRemoveBuilddir struct {
	RepoName string
	BuildID  int
}

// Remove all files for a repo, including all builds, releases, possibly shared home directory.
type msgRemoveRepo struct {
	RepoName string
}

// Remove shared home directory for a repository. Called by explicit user request.
type msgRemoveSharedHome struct {
	RepoName string
}

// Message from unprivileged webserver to root process.
// Only the first non-nil field is handled.
type msg struct {
	Build            *msgBuild
	Chown            *msgChown
	RemoveBuilddir   *msgRemoveBuilddir
	RemoveRepo       *msgRemoveRepo
	RemoveSharedHome *msgRemoveSharedHome
}

// request from one of the http handlers to httpserve's request mux
type request struct {
	msg           msg
	errorResponse chan error
	buildResponse chan buildResult
}

// result of starting a build
type buildResult struct {
	err    error // if non-nil, quick failure.  otherwise, the files below will send updates
	stdout *os.File
	stderr *os.File
	status *os.File // we read a gob-encoded string from status as the exit string
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
