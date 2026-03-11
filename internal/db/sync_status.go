package db

import (
	"database/sql"
	"fmt"
	"time"
)

const initialSyncKey = "initial_sync"

// Check if the initial sync is marked as done in the DB
func (db *DB) IsInitialSyncDone() (bool, error) {
	var value string
	err := db.conn.Get(&value, `SELECT value FROM sync_status WHERE key = $1`, initialSyncKey)
	if err == sql.ErrNoRows {
		return false, nil // not done yet
	} else if err != nil {
		return false, fmt.Errorf("failed to check sync status: %w", err)
	}
	return value == "done", nil
}

// Mark the initial sync as done in the DB
func (db *DB) MarkInitialSyncDone() error {
	_, err := db.conn.Exec(`
		INSERT INTO sync_status (key, value, updated_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
	`, initialSyncKey, "done", time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}
	return nil
}
