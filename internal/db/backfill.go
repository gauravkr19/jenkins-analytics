package db

import (
	"fmt"
	"log"
	"strings"
)

func (db *DB) BackfillEnvColumn() error {
	fmt.Println("Started Backfill")
	
    rows, err := db.conn.Query(`SELECT id, project_path FROM builds WHERE env IS NULL OR env = ''`)
    if err != nil {
        return err
    }
    defer rows.Close()

    type rowData struct {
        ID          int
        ProjectPath string
    }

    var updates []rowData

    for rows.Next() {
        var r rowData
        if err := rows.Scan(&r.ID, &r.ProjectPath); err != nil {
            return err
        }
        updates = append(updates, r)
    }

    for _, r := range updates {
        env := extractEnv(r.ProjectPath)
        _, err := db.conn.Exec(`UPDATE builds SET env = $1 WHERE id = $2`, env, r.ID)
        if err != nil {
            log.Printf("Failed to update env for ID %d: %v", r.ID, err)
        }
    }

    log.Printf("Backfilled env for %d records", len(updates))
    return nil
}

func extractEnv(path string) string {
    parts := strings.Split(path, "/")
    if len(parts) == 0 {
        return "UNKNOWN"
    }

    switch strings.ToUpper(parts[0]) {
    case "DEV":
        return "DEV"
    case "NONPROD", "NON_PROD":
        return "NON_PROD"
    case "PROD_AND_DR", "PROD-DR":
        return "PROD_AND_DR"
    }
    return "UNKNOWN"
}
