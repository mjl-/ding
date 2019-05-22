package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/mjl-/sconf"
)

func TestMain(m *testing.M) {
	check := func(err error, msg string) {
		if err != nil {
			if msg == "" {
				log.Fatal(err)
			}
			log.Fatalf("%s: %s", msg, err)
		}
	}

	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		check(fmt.Errorf("bad command-line arguments, need 1 config file"), "")
	}

	err := sconf.ParseFile(args[0], &config)
	check(err, "parsing config file")
	scripts := parseSQLScripts()

	database, err = sql.Open("postgres", config.Database)
	check(err, "connecting to database")

	tx, err := database.Begin()
	check(err, "begin db transaction")

	_, err = tx.Exec("drop schema if exists public cascade; create schema public")
	check(err, "recreating public schema")

	committing := true
	runScripts(tx, -1, scripts, committing)

	err = tx.Commit()
	check(err, "committing initialized database")

	os.Exit(m.Run())
}
