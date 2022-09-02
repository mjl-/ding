package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func bitbucketHookHandler(w http.ResponseWriter, r *http.Request) {
	if config.BitbucketWebhookSecret == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
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
		log.Printf("bitbucket webhook: invalid secret in request for repoName %s", repoName)
		http.NotFound(w, r)
		return
	}

	// See https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Push
	var event struct {
		Push *struct {
			Changes []struct {
				New *struct {
					Target *struct {
						Type string `json:"type"`
						Hash string `json:"hash"`
					} `json:"target"`
					Name string `json:"name"`
					Type string `json:"type"` // hg: named_branch, tag, bookmark; git: branch, tag
				} `json:"new"` // null for branch deletes
			} `json:"changes"`
		} `json:"push"`
		Repository struct {
			Name string `json:"name"`
			SCM  string `json:"scm"`
		} `json:"repository"`
	}
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		log.Printf("bitbucket webhook: parsing JSON body: %s", err)
		http.Error(w, "bad json", 400)
		return
	}
	if event.Repository.Name != repoName {
		log.Printf("bitbucket webhook: unexpected repoName %s at endpoint for repoName %s", event.Repository.Name, repoName)
		http.Error(w, "bad request", 400)
		return
	}

	var vcs string
	err = database.QueryRowContext(r.Context(), "select vcs from repo where name=$1", repoName).Scan(&vcs)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("bitbucket webhook: reading vcs from database: %s", err)
		http.Error(w, "error", 500)
		return
	}
	if event.Repository.SCM == "hg" && !(vcs == "mercurial" || vcs == "command") {
		log.Printf("bitbucket webhook: misconfigured repository type, bitbucket thinks mercurial, ding thinks %s", vcs)
		http.Error(w, "misconfigured webhook", 500)
		return
	}
	if event.Repository.SCM == "git" && !(vcs == "git" || vcs == "command") {
		log.Printf("bitbucket webhook: misconfigured repository type, bitbucket thinks git, ding thinks %s", vcs)
		http.Error(w, "misconfigured webhook", 500)
		return
	}

	if event.Push == nil {
		http.Error(w, "missing push event", 400)
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
			if vcs == "hg" {
				branch = "default"
			}
		default:
			// we ignore bookmarks
			continue
		}

		if change.New.Target != nil {
			if change.New.Target.Type == "commit" {
				commit := change.New.Target.Hash
				repo, build, buildDir, err := prepareBuild(r.Context(), repoName, branch, commit, false)
				if err != nil {
					log.Printf("bitbucket webhook: error starting build for push event for repo %s, branch %s, commit %s", repoName, branch, commit)
					http.Error(w, "could not create build", 500)
					return
				}
				go doBuild(context.Background(), repo, build, buildDir)
			} else {
				http.Error(w, "New build target is empty", 500)
			}
		}
	}
	w.WriteHeader(204)
}
