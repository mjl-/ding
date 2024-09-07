package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
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
	repoName := r.URL.Path[len("/github/"):]
	if repoName == "" {
		http.NotFound(w, r)
		return
	}

	repo := Repo{Name: repoName}
	settings := Settings{ID: 1}
	err := database.Read(r.Context(), func(tx *bstore.Tx) error {
		err := tx.Get(&repo)
		if err != nil {
			return err
		}
		return tx.Get(&settings)
	})
	if err == bstore.ErrAbsent {
		http.NotFound(w, r)
		return
	} else if err != nil {
		slog.Error("github webhook: reading repo from database", "err", err)
		http.Error(w, "error", http.StatusInternalServerError)
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

	sigOK := func(secret string) bool {
		exp := hmacsha1(secret, buf)
		return hmac.Equal(exp, sig)
	}

	authOK := sigOK(repo.WebhookSecret) || repo.AllowGlobalWebhookSecrets && settings.GithubWebhookSecret != "" && sigOK(settings.GithubWebhookSecret)
	if !authOK {
		slog.Info("github webhook: bad signature, refusing message")
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	if !(repo.VCS == VCSGit || repo.VCS == VCSCommand) {
		slog.Debug("github webhook: push event for a non-git repository")
		http.Error(w, "misconfigured repositories", http.StatusInternalServerError)
		return
	}

	var event githubEvent
	err = json.Unmarshal(buf, &event)
	if err != nil {
		slog.Debug("github webhook: bad JSON body", "err", err)
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	branch := repo.DefaultBranch
	if strings.HasPrefix(event.Ref, "refs/heads/") {
		branch = event.Ref[len("refs/heads/"):]
	}
	commit := event.After
	repo, build, buildDir, gotoolchains, err := prepareBuild(r.Context(), repoName, branch, commit, false)
	if err != nil {
		slog.Error("github webhook: error starting build for push event", "repo", repoName, "branch", branch, "commit", commit)
		http.Error(w, "could not create build", http.StatusInternalServerError)
		return
	}
	go func() {
		err := doBuild(context.Background(), repo, build, buildDir, gotoolchains, false)
		if err != nil {
			slog.Error("build", "err", err)
		}
	}()
	w.WriteHeader(http.StatusNoContent)
}

func hmacsha1(key string, data []byte) []byte {
	hm := hmac.New(sha1.New, []byte(key))
	hm.Write(data)
	return hm.Sum(nil)
}
