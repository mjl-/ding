package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mjl-/bstore"
)

// See https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Push
type bitbucketEvent struct {
	Push *struct {
		Changes []struct {
			New *struct {
				Target *struct {
					Type string `json:"type"`
					Hash string `json:"hash"`
				} `json:"target"`
				Name string `json:"name"`
				Type string `json:"type"` // hg: named_branch, tag, bookmark; git: branch, tag
			} `json:"new"` // Null for branch deletes.
		} `json:"changes"`
	} `json:"push"`
	Repository struct {
		Name string `json:"name"`
		SCM  string `json:"scm"`
	} `json:"repository"`
}

func bitbucketHookHandler(w http.ResponseWriter, r *http.Request) {
	if config.BitbucketWebhookSecret == "" {
		http.NotFound(w, r)
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/bitbucket/") {
		http.NotFound(w, r)
		return
	}
	t := strings.Split(r.URL.Path[len("/bitbucket/"):], "/")
	if len(t) != 2 {
		http.NotFound(w, r)
		return
	}
	repoName := t[0]
	key := t[1]
	if key != config.BitbucketWebhookSecret {
		slog.Info("bitbucket webhook: invalid secret in request", "repo", repoName)
		http.NotFound(w, r)
		return
	}

	var event bitbucketEvent
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		slog.Debug("bitbucket webhook: parsing JSON body", "err", err)
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	repo := Repo{Name: repoName}
	if err := database.Get(r.Context(), &repo); err == bstore.ErrAbsent {
		http.NotFound(w, r)
		return
	} else if err != nil {
		slog.Error("bitbucket webhook: reading repo from database", "err", err)
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	if event.Repository.SCM == "hg" && !(repo.VCS == VCSMercurial || repo.VCS == VCSCommand) {
		slog.Info("bitbucket webhook: misconfigured repository type, got mercurial", "expect", repo.VCS)
		http.Error(w, "misconfigured webhook", http.StatusInternalServerError)
		return
	}
	if event.Repository.SCM == "git" && !(repo.VCS == VCSGit || repo.VCS == VCSCommand) {
		slog.Info("bitbucket webhook: misconfigured repository type, got git", "expect", repo.VCS)
		http.Error(w, "misconfigured webhook", http.StatusInternalServerError)
		return
	}

	if event.Push == nil {
		http.Error(w, "missing push event", http.StatusBadRequest)
		return
	}
	for _, change := range event.Push.Changes {
		if change.New == nil {
			continue
		}
		var branch string
		switch change.New.Type {
		case "branch", "named_branch":
			branch = change.New.Name
		case "tag":
			// todo: fix for silly assumption that people only tag in master/default branch (eg after merge)
			branch = "master"
			if repo.VCS == "hg" {
				branch = "default"
			}
		default:
			// We ignore bookmarks.
			continue
		}

		if change.New.Target != nil {
			if change.New.Target.Type == "commit" {
				commit := change.New.Target.Hash
				repo, build, buildDir, err := prepareBuild(r.Context(), repoName, branch, commit, false)
				if err != nil {
					slog.Error("bitbucket webhook: error starting build for push event", "repo", repoName, "branch", branch, "commit", commit, "err", err)
					http.Error(w, "could not create build", http.StatusInternalServerError)
					return
				}
				go func() {
					err := doBuild(context.Background(), repo, build, buildDir)
					if err != nil {
						slog.Error("build", "err", err)
					}
				}()
			} else {
				http.Error(w, "New build target is empty", http.StatusInternalServerError)
			}
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
