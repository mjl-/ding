package main

import (
	"context"
	"log/slog"
	"net/http"
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

	go func() {
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
	}()

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}
