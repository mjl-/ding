package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"
)

func webhookGoToolchainHandler(w http.ResponseWriter, r *http.Request) {
	settings := Settings{ID: 1}
	err := database.Get(r.Context(), &settings)
	if err != nil {
		http.Error(w, "500 - internal server error - "+err.Error(), http.StatusInternalServerError)
		return
	}
	if settings.GoToolchainWebhookSecret == "" {
		http.Error(w, "401 - unauthorized - no Go Toolchain webhook secret configured", http.StatusUnauthorized)
		return
	} else if r.Header.Get("Authorization") != settings.GoToolchainWebhookSecret {
		http.Error(w, "401 - unauthorized - bad Authorization header", http.StatusUnauthorized)
		return
	}

	var hookData struct {
		Module      string
		Version     string
		LogRecordID int64
		Discovered  time.Time
	}
	if err := json.NewDecoder(r.Body).Decode(&hookData); err != nil {
		http.Error(w, "400 - bad request - parsing json: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Only try updating the toolchain if it is for our platform.
	// Example versions for a toolchain: "v0.0.1-go1.24.1.freebsd-riscv64".
	suffix := "." + runtime.GOOS + "-" + runtime.GOARCH
	if strings.HasSuffix(hookData.Version, suffix) {
		go tryUpdateGoToolchain()
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}

func tryUpdateGoToolchain() {
	defer func() {
		x := recover()
		if x != nil {
			slog.Error("unhandled panic", "err", x)
		}
	}()

	slog.Info("attempting to automatically update go toolchains after webhook")
	m := msg{AutomaticGoToolchain: &msgAutomaticGoToolchain{}}
	err := requestPrivileged(m)
	// Horrible hack, we're passing "updated" as "error" when the toolchains have been
	// updated. todo: change ipc mechanism to properly pass data.
	if err != nil && err.Error() == "updated" {
		if err := scheduleLowPrioBuilds(context.Background(), true); err != nil {
			slog.Error("scheduling low prio builds after updated toolchains", "err", err)
		}
		return
	}
	if err != nil {
		slog.Error("error attempting to update go toolchain", "err", err)
	}
	slog.Info("go toolchains not updated, will try to update again in 15 mins")

	time.Sleep(15 * time.Minute)
	m = msg{AutomaticGoToolchain: &msgAutomaticGoToolchain{}}
	err = requestPrivileged(m)
	if err != nil && err.Error() != "updated" {
		slog.Error("error attempting again to update go toolchain", "err", err)
	} else if err == nil {
		slog.Info("go toolchains not updated in second attempt, giving up for now")
	} else if err := scheduleLowPrioBuilds(context.Background(), true); err != nil {
		slog.Error("scheduling low prio builds after updated toolchains", "err", err)
	}
}
