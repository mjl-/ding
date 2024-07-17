package main

import (
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/mjl-/sconf"
)

type script struct {
	Version  int
	Filename string
	SQL      string
}

func parseSQLScripts() (scripts []script) {
	f, err := httpFS.Open("assets/sql.json")
	check(err, "opening sql scripts")
	check(json.NewDecoder(f).Decode(&scripts), "parsing sql scripts")
	check(f.Close(), "closing sql scripts")

	lastScript := scripts[len(scripts)-1]
	if lastScript.Version != databaseVersion {
		log.Fatalf("databaseVersion %d does not match last upgrade script with version %d", databaseVersion, lastScript.Version)
	}
	return scripts
}

func runScripts(tx *sql.Tx, dbVersion int, scripts []script, committing bool) {
	for _, script := range scripts {
		if script.Version <= dbVersion {
			continue
		}
		_, err := tx.Exec(script.SQL)
		check(err, fmt.Sprintf("executing upgrade script %d: %s: %s", script.Version, script.Filename, err))

		var version int
		err = tx.QueryRow("select max(version) from schema_upgrades").Scan(&version)
		check(err, "fetching database schema version after upgrade")
		if version != script.Version {
			log.Fatalf("invalid upgrade script %s: database not at version %d after running, but at %d", script.Filename, script.Version, version)
		}

		switch script.Version {
		case 5:
			var repoNames []string
			checkRow(tx.QueryRow(`select coalesce(json_agg(name), '[]') from repo`), &repoNames, "reading repos from database")
			for _, repoName := range repoNames {
				dir := fmt.Sprintf("%s/config/%s", config.DataDir, repoName)
				buildSh := readFile(dir + "/build.sh")
				testSh := readFileLax(dir + "/test.sh")
				releaseSh := readFileLax(dir + "/release.sh")
				buildSh += "\n\necho step:test\n" + testSh
				buildSh += "\n\necho step:release\n" + releaseSh
				var id int
				err = tx.QueryRow(`update repo set build_script=$1 where name=$2 returning id`, buildSh, repoName).Scan(&id)
				if err != nil {
					log.Printf("setting repo.build_script for repo %s: %s", repoName, err)
				}
			}

		case 9:
			xerr := func(err, err2 error) error {
				if err == nil {
					return err2
				}
				return err
			}
			q := `
				select coalesce(json_agg(x.*), '[]')
				from (
					select repo.name, build.id
					from repo
					join build on repo.id = build.repo_id
					join release on build.id = release.build_id
				) x
			`
			var repoBuilds []struct {
				Name string
				ID   int
			}
			checkRow(tx.QueryRow(q), &repoBuilds, "reading builds from database")
			for _, repoBuild := range repoBuilds {
				path := fmt.Sprintf("%s/release/%s/%d", config.DataDir, repoBuild.Name, repoBuild.ID)

				files, err := os.ReadDir(path)
				if err != nil {
					log.Printf("upgrade 9, gzipping released files: listing %s: %s (skipping)", path, err)
					continue
				}

				gzipFile := func(file os.DirEntry) {
					opath := path + "/" + file.Name()
					npath := opath + ".gz"
					f, err := os.Open(opath)
					if err != nil {
						log.Printf("upgrade 9, gzipping released files: opening %s: %s (skipping)", opath, err)
						return
					}
					defer f.Close()
					nf, err := os.Create(npath)
					if err != nil {
						log.Printf("upgrade 9, gzipping released files: creating %s: %s (skipping)", npath, err)
						return
					}
					defer func() {
						if nf != nil || !committing {
							err = os.Remove(npath)
							if err != nil {
								log.Printf("upgrade 9, gzipping released files: removing partial new file %s: %s", npath, err)
							}
						} else {
							err = os.Remove(opath)
							if err != nil {
								log.Printf("upgrade 9, gzipping released files: removing old file %s: %s", opath, err)
							}
						}
					}()
					gzw := gzip.NewWriter(nf)
					_, err = io.Copy(gzw, f)
					err = xerr(err, gzw.Close())
					err = xerr(err, nf.Close())
					if err == nil {
						nf = nil
					} else {
						log.Printf("upgrade 9, gzipping released files: gzip %s: %s", opath, err)
						return
					}
				}

				for _, file := range files {
					gzipFile(file)
				}
			}
		case 10:
			var repoBuilds []struct {
				RepoName string
				BuildID  int64
			}
			q := `
				with repo_builds as (
					select
						r.name as repoName,
						b.id as buildID
					from build b
					join repo r on b.repo_id = r.id
				)
				select coalesce(json_agg(rb.*), '[]')
				from repo_builds rb
			`
			checkRow(tx.QueryRow(q), &repoBuilds, "listing builds in database")
			for _, rb := range repoBuilds {
				buildDir := fmt.Sprintf("%s/build/%s/%d/", config.DataDir, rb.RepoName, rb.BuildID)
				du := buildDiskUsage(buildDir)
				qup := `update build set disk_usage=$1 where id=$2 returning id`
				checkRow(tx.QueryRow(qup, du, rb.BuildID), &rb.BuildID, "updating disk usage in database for build")
			}
		}
	}
}

func upgrade(args []string) {
	fs := flag.NewFlagSet("upgrade", flag.ExitOnError)
	dryrun := fs.Bool("dryrun", false, "if set, does rolls back the transaction in which the migration is executed")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: ding upgrade [flags] ding.conf")
		fs.PrintDefaults()
	}
	fs.Parse(args)
	args = fs.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(2)
	}

	err := sconf.ParseFile(args[0], &config)
	check(err, "parsing config file")

	database, err = sql.Open("postgres", config.Database)
	check(err, "connecting to database")

	tx, err := database.Begin()
	check(err, "beginning transaction")

	committing := !*dryrun
	prevDBVersion, newDBVersion := ensureLatestSQL(tx, committing)
	if prevDBVersion == databaseVersion {
		log.Println("database already at latest version", databaseVersion)
		os.Exit(0)
	}

	if committing {
		check(tx.Commit(), "committing")
		log.Printf("database upgrade from %d to %d committed", prevDBVersion, newDBVersion)
	} else {
		check(tx.Rollback(), "rolling back")
		log.Printf("database upgrade from %d to %d rolled back, would succeed", prevDBVersion, newDBVersion)
	}
}

func ensureLatestSQL(tx *sql.Tx, committing bool) (prevDBVersion, newDBVersion int) {
	scripts := parseSQLScripts()
	lastScript := scripts[len(scripts)-1]
	newDBVersion = lastScript.Version

	var have bool
	err := tx.QueryRow("select exists (select 1 from pg_tables where schemaname='public' and tablename='schema_upgrades')").Scan(&have)
	check(err, "checking whether table schema_upgrades exists")

	prevDBVersion = -1
	if have {
		err = tx.QueryRow("select max(version) from schema_upgrades").Scan(&prevDBVersion)
		check(err, "finding database schema version")

		if prevDBVersion == newDBVersion {
			return
		}
	}

	runScripts(tx, prevDBVersion, scripts, committing)
	return
}
