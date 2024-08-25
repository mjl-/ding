package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func hmacsha1(key string, data []byte) []byte {
	hm := hmac.New(sha1.New, []byte(key))
	hm.Write(data)
	return hm.Sum(nil)
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
		BuildScript:   "#!/bin/bash\necho build...\n",
	}
	api.CreateRepo(ctxbg, config.Password, repo)

	ghevent := githubEvent{Ref: "refs/heads/main", After: "e8dab6168e75a88346bc0d2b95ea8227552debf2"}
	ghevent.Repository.Name = "hooktest"
	githubBody := toJSON(ghevent)
	testHook(githubHookHandler, "/github/hooktest", map[string]string{"X-Hub-Signature": fmt.Sprintf("sha1=%x", hmacsha1(config.GithubWebhookSecret, githubBody))}, githubBody, http.StatusNoContent)
	testHook(githubHookHandler, "/github/bogus", map[string]string{"X-Hub-Signature": fmt.Sprintf("sha1=%x", hmacsha1(config.GithubWebhookSecret, githubBody))}, githubBody, http.StatusNotFound)
	testHook(githubHookHandler, "/github/hooktest", map[string]string{}, githubBody, http.StatusBadRequest)
	testHook(githubHookHandler, "/github/hooktest", map[string]string{"X-Hub-Signature": fmt.Sprintf("sha1=%x", hmacsha1(config.GithubWebhookSecret, nil))}, githubBody, http.StatusBadRequest)
	testHook(githubHookHandler, "/github/hooktest", map[string]string{"X-Hub-Signature": fmt.Sprintf("sha1=%x", hmacsha1(config.GithubWebhookSecret, nil))}, nil, http.StatusBadRequest)

	gtevent := githubEvent{Ref: "refs/heads/main", After: "e8dab6168e75a88346bc0d2b95ea8227552debf2"}
	gtevent.Repository.Name = "hooktest"
	giteaBody := toJSON(gtevent)
	testHook(giteaHookHandler, "/gitea/hooktest", map[string]string{"Authorization": "Bearer " + config.GiteaWebhookSecret}, giteaBody, http.StatusNoContent)
	testHook(giteaHookHandler, "/gitea/bogus", map[string]string{"Authorization": "Bearer " + config.GiteaWebhookSecret}, giteaBody, http.StatusNotFound)
	testHook(giteaHookHandler, "/gitea/hooktest", map[string]string{}, giteaBody, http.StatusBadRequest)
	testHook(giteaHookHandler, "/gitea/hooktest", map[string]string{"Authorization": "Bearer bogus"}, giteaBody, http.StatusBadRequest)
	testHook(giteaHookHandler, "/gitea/hooktest", map[string]string{"Authorization": "Bearer " + config.GiteaWebhookSecret}, nil, http.StatusBadRequest)

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
	testHook(bitbucketHookHandler, "/bitbucket/hooktest/"+config.BitbucketWebhookSecret, nil, bitbucketBody, http.StatusNoContent)
	testHook(bitbucketHookHandler, "/bitbucket/bogus/"+config.BitbucketWebhookSecret, nil, bytes.ReplaceAll(bitbucketBody, []byte("hooktest"), []byte("bogus")), http.StatusNotFound)
	testHook(bitbucketHookHandler, "/bitbucket/hooktest/bogus", nil, bitbucketBody, http.StatusNotFound)

	time.Sleep(200 * time.Millisecond) // todo: properly wait for builds to fail.
}
