package main

import (
	"encoding/json"
)

type eventStringer interface {
	eventString() (string, []byte, error)
}

// EventRepo represents an update of a repository or creation of a repository.
type EventRepo struct {
	Repo Repo
}

func (e EventRepo) eventString() (string, []byte, error) {
	buf, err := json.Marshal(e)
	return "repo", buf, err
}

// EventRemoveRepo represents the removal of a repository.
type EventRemoveRepo struct {
	RepoName string
}

func (e EventRemoveRepo) eventString() (string, []byte, error) {
	buf, err := json.Marshal(e)
	return "removeRepo", buf, err
}

// EventBuild represents an update to a build, or the start of a new build.
// Output is not part of the build, see EventOutput below.
type EventBuild struct {
	RepoName string
	Build    Build
}

func (e EventBuild) eventString() (string, []byte, error) {
	buf, err := json.Marshal(e)
	return "build", buf, err
}

// EventRemoveBuild represents the removal of a build from the database.
type EventRemoveBuild struct {
	RepoName string
	BuildID  int32
}

func (e EventRemoveBuild) eventString() (string, []byte, error) {
	buf, err := json.Marshal(e)
	return "removeBuild", buf, err
}

// EventOutput represents new output from a build.
// Text only contains the newly added output, not the full output so far.
type EventOutput struct {
	BuildID int32
	Step    string // During which the output was generated, eg `clone`, `build`.
	Where   string // `stdout` or `stderr`.
	Text    string // Lines of text written.
}

func (e EventOutput) eventString() (string, []byte, error) {
	buf, err := json.Marshal(e)
	return "output", buf, err
}
