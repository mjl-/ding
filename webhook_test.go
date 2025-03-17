package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"
)

func toJSON(v any) []byte {
	buf, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return buf
}

func TestWebhookGoToolchainAuth(t *testing.T) {
	testHook := func(h http.HandlerFunc, path string, headers map[string]string, body []byte, expCode int) {
		t.Helper()

		w := httptest.NewRecorder()
		w.Body = &bytes.Buffer{}
		r := httptest.NewRequest("POST", path, bytes.NewReader(body))
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		h(w, r)
		if w.Code != expCode {
			t.Fatalf("got code %d, expected %d, body %q", w.Code, expCode, w.Body.String())
		}
	}

	testEnv(t)

	// No secret configured.
	testHook(webhookGoToolchainHandler, "/gotoolchain", nil, nil, http.StatusUnauthorized)
	testHook(webhookGoToolchainHandler, "/gotoolchain", map[string]string{"Authorization": "bogus"}, nil, http.StatusUnauthorized)

	settings := Settings{ID: 1}
	err := database.Get(ctxbg, &settings)
	tcheck(t, err, "get settings")
	settings.GoToolchainWebhookSecret = "Bearer " + genSecret()
	err = database.Update(ctxbg, &settings)
	tcheck(t, err, "save settings with go toolchains webhook secret")

	testHook(webhookGoToolchainHandler, "/gotoolchain", nil, nil, http.StatusUnauthorized)
	testHook(webhookGoToolchainHandler, "/gotoolchain", map[string]string{"Authorization": "bogus"}, nil, http.StatusUnauthorized)

	headers := map[string]string{"Authorization": settings.GoToolchainWebhookSecret, "Content-Type": "application/json"}

	// Missing JSON body.
	testHook(webhookGoToolchainHandler, "/gotoolchain", headers, nil, http.StatusBadRequest)

	hookData := struct {
		Module      string
		Version     string
		LogRecordID int64
		Discovered  time.Time
	}{"golang.org/toolchain", "", 1, time.Now()}
	buf, err := json.Marshal(hookData)
	tcheck(t, err, "marshal json")

	// No matching GOOS/GOARCH, so no toolchain fetch.
	testHook(webhookGoToolchainHandler, "/gotoolchain", headers, buf, http.StatusOK)

	if os.Getenv("DING_TEST_GOTOOLCHAINS") == "" {
		t.Skip("skipping because DING_TEST_GOTOOLCHAINS is not set")
	}

	hookData.Version = "v0.0.1-go1.24.1." + runtime.GOOS + "-" + runtime.GOARCH
	buf, err = json.Marshal(hookData)
	tcheck(t, err, "marshal json")

	testHook(webhookGoToolchainHandler, "/gotoolchain", headers, buf, http.StatusOK)
}

func TestWebhook(t *testing.T) {
	testHook := func(h http.HandlerFunc, path string, headers map[string]string, body []byte, expCode int) {
		t.Helper()

		w := httptest.NewRecorder()
		w.Body = &bytes.Buffer{}
		r := httptest.NewRequest("POST", path, bytes.NewReader(body))
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		h(w, r)
		if w.Code != expCode {
			t.Fatalf("got code %d, expected %d, body %q", w.Code, expCode, w.Body.String())
		}
	}

	testEnv(t)
	api := Ding{}

	repo := Repo{
		Name:          "hooktest",
		VCS:           VCSCommand,
		Origin:        "sh -c 'echo clone..; mkdir -p checkout/$DING_CHECKOUTPATH; echo commit: ...'",
		DefaultBranch: "main",
		CheckoutPath:  "hooktest",
		BuildScript:   "#!/usr/bin/env bash\necho build...\n",
	}
	repo = api.RepoCreate(ctxbg, config.Password, repo)

	ghevent := githubEvent{Ref: "refs/heads/main", After: "e8dab6168e75a88346bc0d2b95ea8227552debf2"}
	ghevent.Repository.Name = "hooktest"
	githubBody := toJSON(ghevent)
	testHook(githubHookHandler, "/github/hooktest", map[string]string{"X-Hub-Signature": fmt.Sprintf("sha1=%x", hmacsha1(repo.WebhookSecret, githubBody))}, githubBody, http.StatusNoContent)
	testHook(githubHookHandler, "/github/bogus", map[string]string{"X-Hub-Signature": fmt.Sprintf("sha1=%x", hmacsha1(repo.WebhookSecret, githubBody))}, githubBody, http.StatusNotFound)
	testHook(githubHookHandler, "/github/hooktest", map[string]string{}, githubBody, http.StatusBadRequest)
	testHook(githubHookHandler, "/github/hooktest", map[string]string{"X-Hub-Signature": fmt.Sprintf("sha1=%x", hmacsha1(repo.WebhookSecret, nil))}, githubBody, http.StatusBadRequest)
	testHook(githubHookHandler, "/github/hooktest", map[string]string{"X-Hub-Signature": fmt.Sprintf("sha1=%x", hmacsha1(repo.WebhookSecret, nil))}, nil, http.StatusBadRequest)

	gtevent := githubEvent{Ref: "refs/heads/main", After: "e8dab6168e75a88346bc0d2b95ea8227552debf2"}
	gtevent.Repository.Name = "hooktest"
	giteaBody := toJSON(gtevent)
	testHook(giteaHookHandler, "/gitea/hooktest", map[string]string{"Authorization": "Bearer " + repo.WebhookSecret}, giteaBody, http.StatusNoContent)
	testHook(giteaHookHandler, "/gitea/bogus", map[string]string{"Authorization": "Bearer " + repo.WebhookSecret}, giteaBody, http.StatusNotFound)
	testHook(giteaHookHandler, "/gitea/hooktest", map[string]string{}, giteaBody, http.StatusBadRequest)
	testHook(giteaHookHandler, "/gitea/hooktest", map[string]string{"Authorization": "Bearer bogus"}, giteaBody, http.StatusBadRequest)
	testHook(giteaHookHandler, "/gitea/hooktest", map[string]string{"Authorization": "Bearer " + repo.WebhookSecret}, nil, http.StatusBadRequest)

	bitbucketBody := []byte(`
{
	"repository": {
		"name": "hooktest",
		"scm": "git"
	},
	"push": {
		"changes": [
			{
				"new": {
					"target": {
						"type": "commit",
						"hash": "e8dab6168e75a88346bc0d2b95ea8227552debf2"
					},
					"name": "main",
					"type": "branch"
				}
			}
		]
	}
}`)
	testHook(bitbucketHookHandler, "/bitbucket/hooktest/"+repo.WebhookSecret, nil, bitbucketBody, http.StatusNoContent)
	testHook(bitbucketHookHandler, "/bitbucket/bogus/"+repo.WebhookSecret, nil, bytes.ReplaceAll(bitbucketBody, []byte("hooktest"), []byte("bogus")), http.StatusNotFound)
	testHook(bitbucketHookHandler, "/bitbucket/hooktest/bogus", nil, bitbucketBody, http.StatusNotFound)

	time.Sleep(200 * time.Millisecond) // todo: properly wait for builds to fail.
}
