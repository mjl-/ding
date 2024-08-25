package main

import (
	"github.com/mjl-/bstore"
)

func _removeBuild(tx *bstore.Tx, repoName string, buildID int32) {
	b := Build{ID: buildID}
	err := tx.Get(&b)
	_checkf(err, "get build to remove")
	err = tx.Delete(&b)
	_checkf(err, "remove build from database")

	if !b.BuilddirRemoved {
		_removeBuildDir(b)
	}
}

func _removeBuildDir(b Build) {
	msg := msg{RemoveBuilddir: &msgRemoveBuilddir{b.RepoName, b.ID}}
	err := requestPrivileged(msg)
	_checkf(err, "removing build dir")
}
