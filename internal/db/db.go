package db

// Connect to PostgreSQL
// Provide a function to insert Build metadata
// Use github.com/jmoiron/sqlx for simpler DB access with structs
import (
	"fmt"

	"github.com/gauravkr19/jenkins-analytics/models"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DB struct {
	conn *sqlx.DB
}

func NewDB(dsn string) (*DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("could not connect to db: %w", err)
	}
	return &DB{conn: db}, nil
}

func (db *DB) InsertBuild(b *models.Build) error {
	query := `
	INSERT INTO builds (
		build_number, project_name, user_id, status, result,
		timestamp, duration_ms, branch, commit_id, job_url,
		console_log_head, console_log_tail, error_message
	)
	VALUES (
		:build_number, :project_name, :user_id, :status, :result,
		:timestamp, :duration_ms, :branch, :commit_id, :job_url,
		:console_log_head, :console_log_tail, :error_message
	)
	RETURNING id
	`
	rows, err := db.conn.NamedQuery(query, b)
	if err != nil {
		return fmt.Errorf("insert build failed: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&b.ID); err != nil {
			return fmt.Errorf("could not scan returned ID: %w", err)
		}
	}

	return nil
}

func (db *DB) GetBuildByID(id int) (*models.Build, error) {
	var build models.Build
	err := db.conn.Get(&build, `SELECT * FROM builds WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("get build by id failed: %w", err)
	}
	return &build, nil
}
