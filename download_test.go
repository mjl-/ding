package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownload(t *testing.T) {
	testEnv(t)
	api := Ding{}

	const buildScript = `#!/bin/bash
set -e
echo building...
echo hi>myfile
echo version: 1.2.3
echo release: mycmd linux amd64 toolchain1.2.3 myfile
touch $DING_DOWNLOADDIR/coverage.txt
echo coverage: 80.0
echo coverage-report: coverage.txt
`

	r := Repo{
		Name:          "dltest",
		VCS:           VCSCommand,
		Origin:        "sh -c 'echo clone..; mkdir -p checkout/$DING_CHECKOUTPATH; echo commit: ...'",
		DefaultBranch: "main",
		CheckoutPath:  "dltest",
		BuildScript:   buildScript,
	}
	api.CreateRepo(ctxbg, config.Password, r)

	b := api.CreateBuild(ctxbg, config.Password, r.Name, "unused", "")
	twaitBuild(t, b, StatusSuccess)

	testGet := func(h http.HandlerFunc, path string, expCode int) {
		t.Helper()

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)
		h(w, r)
		tcompare(t, w.Code, expCode)
	}

	testGet(serveDownload, fmt.Sprintf("/dl/file/dltest/%d/coverage.txt", b.ID), http.StatusOK)
	testGet(serveDownload, fmt.Sprintf("/dl/file/dltest/%d/bogus.txt", b.ID), http.StatusNotFound)
	testGet(serveDownload, fmt.Sprintf("/dl/file/bogus/%d/coverage.txt", b.ID), http.StatusNotFound)
	testGet(serveDownload, fmt.Sprintf("/dl/file/dltest/%d/coverage.txt", b.ID+990), http.StatusNotFound)
	testGet(serveDownload, fmt.Sprintf("/dl/result/dltest/%d/any.zip", b.ID), http.StatusOK)
	testGet(serveDownload, fmt.Sprintf("/dl/result/dltest/%d/any.tgz", b.ID), http.StatusOK)
	testGet(serveDownload, fmt.Sprintf("/dl/release/dltest/%d/any.zip", b.ID), http.StatusNotFound)
	testGet(serveDownload, fmt.Sprintf("/dl/release/dltest/%d/any.tgz", b.ID), http.StatusNotFound)

	testGet(serveResult, fmt.Sprintf("/result/dltest/%d/myfile", b.ID), http.StatusOK)
	testGet(serveResult, fmt.Sprintf("/result/bogus/%d/myfile", b.ID), http.StatusNotFound)
	testGet(serveResult, fmt.Sprintf("/result/dltest/%d/myfile", b.ID+999), http.StatusNotFound)
	testGet(serveResult, fmt.Sprintf("/result/dltest/%d/bogusfile", b.ID), http.StatusNotFound)
	testGet(serveRelease, fmt.Sprintf("/release/dltest/%d/myfile", b.ID), http.StatusNotFound)

	api.CreateRelease(ctxbg, config.Password, r.Name, b.ID)
	testGet(serveDownload, fmt.Sprintf("/dl/release/dltest/%d/any.zip", b.ID), http.StatusOK)
	testGet(serveDownload, fmt.Sprintf("/dl/release/dltest/%d/any.tgz", b.ID), http.StatusOK)
	testGet(serveDownload, fmt.Sprintf("/dl/result/dltest/%d/any.zip", b.ID), http.StatusOK)
	testGet(serveDownload, fmt.Sprintf("/dl/result/dltest/%d/any.tgz", b.ID), http.StatusOK)
	testGet(serveDownload, fmt.Sprintf("/dl/release/dltest/%d/any.bogus", b.ID), http.StatusNotFound)
	testGet(serveDownload, fmt.Sprintf("/dl/result/dltest/%d/any.bogus", b.ID), http.StatusNotFound)

	testGet(serveResult, fmt.Sprintf("/result/dltest/%d/myfile", b.ID), http.StatusOK)
	testGet(serveRelease, fmt.Sprintf("/release/dltest/%d/myfile", b.ID), http.StatusOK)
}
