package main

import (
	"database/sql"
)

func _removeBuild(tx *sql.Tx, repoName string, buildID int) {
	_, err := tx.Exec(`delete from result where build_id=$1`, buildID)
	sherpaCheck(err, "removing results from database")

	builddirRemoved := false
	q := `delete from build where id=$1 returning builddir_removed`
	sherpaCheckRow(tx.QueryRow(q, buildID), &builddirRemoved, "removing build from database")

	if !builddirRemoved {
		msg := msg{RemoveBuilddir: &msgRemoveBuilddir{repoName, buildID}}
		err := requestPrivileged(msg)
		sherpaCheck(err, "removing build dir")
	}
}
