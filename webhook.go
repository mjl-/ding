package main

import (
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
		updated, err := automaticGoToolchain()
		if err == nil && updated {
			return
		}
		if err != nil {
			slog.Error("error attempting to update go toolchain", "err", err)
		}
		slog.Info("go toolchains not updated, will try to update again in 15 mins")
		time.Sleep(15 * time.Minute)
		updated, err = automaticGoToolchain()
		if err != nil {
			slog.Error("error attempting again to update go toolchain", "err", err)
		} else if !updated {
			slog.Info("go toolchains not updated in second attempt, giving up for now")
		}
	}()

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}
