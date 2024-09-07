package main

import (
	"log"
	"log/slog"
	"os"
	"path"
)

func getgo() {
	loglevel.Set(slog.LevelDebug)
	config.GoToolchainDir = path.Join(os.Getenv("HOME"), "sdk")

	active := activeGoToolchains()
	updated, err := automaticGoToolchain()
	if err != nil {
		log.Fatalf("error: %v", err)
	} else if updated {
		nactive := activeGoToolchains()
		slog.Info("updated", "from", active, "to", nactive)
	}
}
