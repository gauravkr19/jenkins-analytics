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

func (db *DB) FetchRecentBuilds(limit int) ([]*models.Build, error) {
	query := `
		SELECT id, build_number, project_name, project_path, user_id, status,
		       timestamp, duration_ms, branch, job_url
		FROM builds
		ORDER BY timestamp DESC
		LIMIT $1
	`

	var builds []*models.Build
	if err := db.conn.Select(&builds, query, limit); err != nil {
		return nil, fmt.Errorf("fetch recent builds failed: %w", err)
	}
	return builds, nil
}

// ALTER TABLE builds ADD CONSTRAINT unique_build_path UNIQUE (build_number, project_path);
func (db *DB) InsertBuild(b *models.Build) error {
	query := `
	INSERT INTO builds (
		build_number, project_name, project_path, user_id, status,
		timestamp, duration_ms, branch, job_url
	)
	VALUES (
		:build_number, :project_name, :project_path, :user_id, :status,
		:timestamp, :duration_ms, :branch, :job_url,
	)
	ON CONFLICT (build_number, project_path) DO NOTHING
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

func (db *DB) GetAllBuilds() ([]*models.Build, error) {
	query := `SELECT build_number, project_name, job_url FROM builds`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var builds []*models.Build
	for rows.Next() {
		var b models.Build
		if err := rows.Scan(&b.BuildNumber, &b.ProjectName, &b.JobURL); err != nil {
			return nil, err
		}
		builds = append(builds, &b)
	}
	return builds, nil
}
