package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/mjl-/bstore"
)

type githubEvent struct {
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	Ref   string `json:"ref"`
	After string `json:"after"`
}

func githubHookHandler(w http.ResponseWriter, r *http.Request) {
	if config.GithubWebhookSecret == "" {
		http.NotFound(w, r)
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/github/") {
		http.NotFound(w, r)
		return
	}
	repoName := r.URL.Path[len("/github/"):]

	repo := Repo{Name: repoName}
	if err := database.Get(r.Context(), &repo); err == bstore.ErrAbsent {
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Printf("github webhook: reading repo from database: %s", err)
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	if !(repo.VCS == VCSGit || repo.VCS == VCSCommand) {
		log.Printf("github webhook: push event for a non-git repository")
		http.Error(w, "misconfigured repositories", http.StatusInternalServerError)
		return
	}

	sigstr := strings.TrimSpace(r.Header.Get("X-Hub-Signature"))
	t := strings.Split(sigstr, "=")
	if len(t) != 2 || t[0] != "sha1" || len(t[1]) != 2*sha1.Size {
		http.Error(w, "malformed/missing X-Hub-Signature header", http.StatusBadRequest)
		return
	}
	sig, err := hex.DecodeString(t[1])
	if err != nil {
		http.Error(w, "malformed hex in X-Hub-Signature", http.StatusBadRequest)
		return
	}
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "error reading request", http.StatusInternalServerError)
		return
	}
	mac := hmac.New(sha1.New, []byte(config.GithubWebhookSecret))
	mac.Write(buf)
	exp := mac.Sum(nil)
	if !hmac.Equal(exp, sig) {
		log.Printf("github webhook: bad signature, refusing message")
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}
	var event githubEvent
	err = json.Unmarshal(buf, &event)
	if err != nil {
		log.Printf("github webhook: bad JSON body: %s", err)
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if event.Repository.Name != repoName {
		log.Printf("github webhook: repository does not match, github sent %s for URL for %s", event.Repository.Name, repoName)
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
		log.Printf("github webhook: error starting build for push event for repo %s, branch %s, commit %s", repoName, branch, commit)
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
