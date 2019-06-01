package main

import (
	"context"
	"database/sql"
	"log"
)

func transact(ctx context.Context, fn func(tx *sql.Tx)) {
	tx, err := database.BeginTx(ctx, nil)
	sherpaCheck(err, "starting database transaction")

	defer func() {
		if tx != nil {
			ee := tx.Rollback()
			if ee != nil {
				log.Println("rolling back:", ee)
			}
		}
	}()
	fn(tx)
	err = tx.Commit()
	tx = nil
	sherpaCheck(err, "committing database transaction")
}
