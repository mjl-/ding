package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
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
		log.Printf("gitea webhook: reading repo from database: %s", err)
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	if !(repo.VCS == VCSGit || repo.VCS == VCSCommand) {
		log.Printf("gitea webhook: push event for a non-git repository")
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
		log.Printf("gitea webhook: bad JSON body: %s", err)
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if event.Repository.Name != repoName {
		log.Printf("gitea webhook: repository does not match, gitea sent %s for URL for %s", event.Repository.Name, repoName)
		http.Error(w, "repository mismatch", http.StatusBadRequest)
		return
	}
	branch := repo.DefaultBranch
	if strings.HasPrefix(event.Ref, "refs/heads/") {
		branch = event.Ref[len("refs/heads/"):]
	}
	commit := event.After
	repo, build, buildDir, err := prepareBuild(r.Context(), repoName, branch, commit, false)
	if err != nil {
		log.Printf("gitea webhook: error starting build for push event for repo %s, branch %s, commit %s", repoName, branch, commit)
		http.Error(w, "could not create build", http.StatusInternalServerError)
		return
	}
	go func() {
		err := doBuild(context.Background(), repo, build, buildDir)
		if err != nil {
			log.Printf("build: %s", err)
		}
	}()
	w.WriteHeader(http.StatusNoContent)
}
