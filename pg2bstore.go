package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/mjl-/bstore"
)

func migratePostgresToBstore(dbPath string) {
	pgdb, err := sql.Open("postgres", config.Database)
	xcheckf(err, "opening original database connection")
	defer func() {
		err := pgdb.Close()
		xcheckf(err, "closing postgres connection")
	}()

	var pgdbVersion int
	err = pgdb.QueryRow("select max(version) from schema_upgrades").Scan(&pgdbVersion)
	xcheckf(err, "fetching database schema version")
	if pgdbVersion != 18 {
		log.Fatalf("bad database schema version, expected 18, saw %d, upgrade install to version before this to upgrade database schema", pgdbVersion)
	}

	var repos map[int32]Repo
	const qrepo = `
		select json_object_agg(id, r.*)
		from repo r
	`
	xcheckRow(pgdb.QueryRow(qrepo), &repos, "retrieving original repos")

	type OBuild struct {
		RepoID int32 `json:"repo_id"`
		Build
	}
	var builds []OBuild
	const qbuilds = `
		select coalesce(json_agg(x.*), '[]')
		from (
			select * from build_with_result
		) x
	`
	xcheckRow(pgdb.QueryRow(qbuilds), &builds, "retrieving original builds with results")

	type ORelease struct {
		// Field Time is ignored, we already have it in Build.
		Steps       []Step `json:"steps"`
		BuildScript string `json:"build_script"`
	}
	var releases map[int32]ORelease
	const qrel = `
		select coalesce(json_object_agg(build_id, x.*), '[]')
		from (
			select * from release
		) x
	`
	xcheckRow(pgdb.QueryRow(qrel), &releases, "retrieving original releases")

	opts := bstore.Options{Timeout: 5 * time.Second}
	bdb, err := bstore.Open(context.Background(), dbPath, &opts, Repo{}, Build{})
	xcheckf(err, "open target database")
	defer func() {
		err := bdb.Close()
		xcheckf(err, "closing bstore database")
	}()

	lcheckf := func(err error, format string, args ...any) {
		if err != nil {
			log.Printf(format, args...)
			if xerr := os.Remove(dbPath); xerr != nil {
				log.Printf("removing new database path, must clean up manually before retrying: %v", xerr)
			}
			os.Exit(1)
		}
	}

	err = bdb.Write(context.Background(), func(tx *bstore.Tx) error {
		for _, r := range repos {
			err := tx.Insert(&r)
			lcheckf(err, "insert repo")
		}

		for _, ob := range builds {
			b := ob.Build
			repo := repos[ob.RepoID]
			b.RepoName = repo.Name
			if b.RepoName == "" {
				lcheckf(fmt.Errorf("unknown repo for build %d", b.ID), "looking up repo for build")
			}
			if rel, ok := releases[b.ID]; ok {
				b.Steps = rel.Steps
				b.BuildScript = rel.BuildScript
			} else if !b.BuilddirRemoved {
				buildDir := fmt.Sprintf("%s/build/%s/%d", dingDataDir, repo.Name, b.ID)
				buildSh := buildDir + "/scripts/build.sh"
				buf, err := os.ReadFile(buildSh)
				if err != nil {
					log.Printf("read build.sh: %v (skipped)", err)
				}
				b.BuildScript = string(buf)
			}
			err := tx.Insert(&b)
			lcheckf(err, "insert build")
		}

		return nil
	})
	lcheckf(err, "target db transaction")
}

func xcheckRow(row *sql.Row, r any, msg string) {
	var buf []byte
	err := row.Scan(&buf)
	if err == sql.ErrNoRows {
		log.Fatal("no row in result")
	}
	xcheckf(err, "%s: reading json from database row into buffer", msg)
	err = json.Unmarshal(buf, r)
	xcheckf(err, "%s: parsing json from database", msg)
}
