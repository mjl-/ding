package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mjl-/bstore"
)

type giteaEvent struct {
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	Ref   string `json:"ref"`
	After string `json:"after"`
}

func giteaHookHandler(w http.ResponseWriter, r *http.Request) {
	if config.GiteaWebhookSecret == "" {
		http.NotFound(w, r)
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/gitea/") {
		http.NotFound(w, r)
		return
	}
	repoName := r.URL.Path[len("/gitea/"):]

	repo := Repo{Name: repoName}
	if err := database.Get(r.Context(), &repo); err == bstore.ErrAbsent {
		http.NotFound(w, r)
		return
	} else if err != nil {
		slog.Error("gitea webhook: reading repo from database", "err", err)
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	if !(repo.VCS == VCSGit || repo.VCS == VCSCommand) {
		slog.Debug("gitea webhook: push event for a non-git repository")
		http.Error(w, "misconfigured repositories", http.StatusInternalServerError)
		return
	}

	authHdr := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHdr != "Bearer "+config.GiteaWebhookSecret {
		http.Error(w, "invalid/missing authorization header", http.StatusBadRequest)
		return
	}

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error reading request", http.StatusInternalServerError)
		return
	}
	var event giteaEvent
	err = json.Unmarshal(buf, &event)
	if err != nil {
		slog.Debug("gitea webhook: bad JSON body", "err", err)
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	branch := repo.DefaultBranch
	if strings.HasPrefix(event.Ref, "refs/heads/") {
		branch = event.Ref[len("refs/heads/"):]
	}
	commit := event.After
	repo, build, buildDir, err := prepareBuild(r.Context(), repoName, branch, commit, false)
	if err != nil {
		slog.Error("gitea webhook: error starting build for push event", "repo", repoName, "branch", branch, "commit", commit, "err", err)
		http.Error(w, "could not create build", http.StatusInternalServerError)
		return
	}
	go func() {
		err := doBuild(context.Background(), repo, build, buildDir)
		if err != nil {
			slog.Error("build", "err", err)
		}
	}()
	w.WriteHeader(http.StatusNoContent)
}
