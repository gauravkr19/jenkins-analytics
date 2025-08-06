package db

// Connect to PostgreSQL
// Provide a function to insert Build metadata
// Use github.com/jmoiron/sqlx for simpler DB access with structs
import (
	"fmt"
	"log"
	"strings"
	"time"

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
		timestamp, duration_ms, job_url, branch, git_url, commit_sha, deploy_env, trigger_type, env
	)
	VALUES (
		:build_number, :project_name, :project_path, :user_id, :status,
		:timestamp, :duration_ms, :job_url, :branch, :git_url, :commit_sha, :deploy_env, :trigger_type, :env
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

// GetBuildsByTime fetches builds in [from, to], sorted and paginated.
func (db *DB) GetBuildsByTime(from, to time.Time,limit, offset int,sortBy, order string) ([]models.Build, error) {
    // Whitelist allowed sort columns to avoid SQL injection
    // whitelist allowed sort columns -> actual DB column names
    allowed := map[string]string{
        "timestamp":   "timestamp",
        "env":         "env",
        "status":      "status",
        "user_id":     "user_id",
        "duration_ms": "duration_ms",
    }
    col, ok := allowed[sortBy]
    if !ok {
        col = "timestamp"
    }

    // normalize order
    if strings.ToUpper(order) == "ASC" {
        order = "ASC"
    } else {
        order = "DESC"
    }

    // Explicit column list (must match your models.Build tags)
    cols := []string{
        "id",
        "build_number",
        "project_name",
        "project_path",
        "user_id",
        "status",
        "timestamp",
        "duration_ms",
        "job_url",
        "branch",
        "git_url",
        "commit_sha",
        "deploy_env",
        "trigger_type",
        "env",
    }
    colList := strings.Join(cols, ", ")

    // build query with dynamic ORDER BY
    query := fmt.Sprintf(`
        SELECT %s
        FROM builds
        WHERE timestamp BETWEEN $1 AND $2
        ORDER BY %s %s
        LIMIT $3 OFFSET $4
    `, colList, col, order)

    var builds []models.Build
    if err := db.conn.Select(&builds, query, from, to, limit, offset); err != nil {
        return nil, fmt.Errorf("GetBuildsByTime: %w", err)
    }
    return builds, nil
}

// count query for pagination
func (db *DB) CountBuildsByTime(from, to time.Time) (int, error) {
	var count int
	err := db.conn.Get(&count, `
        SELECT COUNT(*) FROM builds
        WHERE timestamp BETWEEN $1 AND $2
    `, from, to)
	if err != nil {
		return 0, fmt.Errorf("failed to count builds: %w", err)
	}
	return count, nil
}

// Used during incremental fetch of build records
func (db *DB) GetLastSeenBuildNumber(projectName string) (int, error) {
	var lastSeen int
	err := db.conn.QueryRow(`
        SELECT COALESCE(MAX(build_number), 0) FROM builds WHERE project_name = $1
    `, projectName).Scan(&lastSeen)
	return lastSeen, err
}

// Uses project path to construct hierarchy of builds for each folder.
func (db *DB) GetBuildsByFolder() (map[string]map[string][]string, error) {
	paths, err := db.GetAllProjectPaths()
	if err != nil {
		return nil, err
	}

	tree := make(map[string]map[string][]string)

	for _, path := range paths {
		parts := strings.Split(path, "/")
		if len(parts) != 3 {
			continue // skip malformed path
		}

		folder, app, pipeline := parts[0], parts[1], parts[2]
		if _, ok := tree[folder]; !ok {
			tree[folder] = make(map[string][]string)
		}
		tree[folder][app] = appendIfMissing(tree[folder][app], pipeline)
	}

	return tree, nil
}

// Returns slice of pipelines to construct tree
func appendIfMissing(slice []string, val string) []string {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}

// Retrieves the project paths
func (db *DB) GetAllProjectPaths() ([]string, error) {
	rows, err := db.conn.Query(`SELECT DISTINCT project_path FROM builds`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}

	return paths, nil
}

// Used by ExportBuildsToExcel
func (db *DB) GetBuildsByProject(projectPath string) ([]models.Build, error) {
    var builds []models.Build
    err := db.conn.Select(&builds, `
        SELECT * FROM builds
        WHERE project_path = $1
        ORDER BY timestamp DESC
    `, projectPath)
    return builds, err
}

func (db *DB) GetBuildsByProjectPath(path string) ([]models.Build, error) {
	rows, err := db.conn.Query(`
        SELECT build_number, env, project_path, status, user_id,
               timestamp, duration_ms, job_url, trigger_type, git_url, branch, commit_sha
        FROM builds
        WHERE project_path = $1
        ORDER BY build_number DESC
        LIMIT 100
    `, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var builds []models.Build
	for rows.Next() {
		var b models.Build
		err := rows.Scan(
			&b.BuildNumber,
			&b.Env,
			&b.ProjectPath,
			&b.Status,
			&b.UserID,
			&b.Timestamp,
			&b.DurationMS,
			&b.JobURL,
			&b.TriggerType,
			&b.GitRepo,
			&b.Branch,
			&b.CommitSHA,
		)
		if err != nil {
			return nil, err
		}
		builds = append(builds, b)
	}

	return builds, nil
}

func (db *DB) GetBuildTree() (*models.FolderNode, error) {
	paths, err := db.GetAllProjectPaths()
	if err != nil {
		return nil, err
	}

	root := &models.FolderNode{
		Name:     "root",
		FullPath: "",
		Children: map[string]*models.FolderNode{},
	}

	for _, path := range paths {
		parts := strings.Split(path, "/")
		curr := root
		currPath := ""

		for i, part := range parts {
			currPath = strings.TrimLeft(currPath+"/"+part, "/")

			if curr.Children == nil {
				curr.Children = make(map[string]*models.FolderNode)
			}

			child, exists := curr.Children[part]
			if !exists {
				child = &models.FolderNode{
					Name:     part,
					FullPath: currPath,
					Children: map[string]*models.FolderNode{},
				}
				curr.Children[part] = child
			}

			curr = child

			if i == len(parts)-1 {
				curr.IsLeaf = true
			}
		}
	}

	return root, nil
}

func (db *DB) GetRecentBuildsMissingStatus(limit int) ([]*models.Build, error) {
    rows, err := db.conn.Query(`
        SELECT id, build_number, job_url
        FROM builds
        WHERE (status IS NULL OR status = '')
        ORDER BY timestamp DESC
        LIMIT $1
    `, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var builds []*models.Build
    for rows.Next() {
        var b models.Build
        err := rows.Scan(&b.ID, &b.BuildNumber, &b.JobURL)
        if err != nil {
            return nil, err
        }
        builds = append(builds, &b)
    }

    return builds, nil
}

// Patches missing Status field
func (db *DB) UpdateBuildStatus(id int, status string) error {
    _, err := db.conn.Exec(`
        UPDATE builds SET status = $1 WHERE id = $2
    `, status, id)
    return err
}

// CleanupToMax deletes the oldest builds so that only cfg.MaxRecords remain.
func (db *DB) CleanupToMax(maxRecords, deleteMultiple int) error {
    // 1) Total before
    var totalBefore int
    if err := db.conn.QueryRow(`SELECT count(*) FROM builds`).Scan(&totalBefore); err != nil {
        return fmt.Errorf("count before delete failed: %w", err)
    }
    log.Printf("[Cleaner] totalBefore=%d, maxRecords=%d", totalBefore, maxRecords)
    if totalBefore <= maxRecords {
        return nil
    }

    // 2) Oldest timestamp
    var oldest time.Time
    if err := db.conn.QueryRow(`SELECT min("timestamp") FROM builds`).Scan(&oldest); err != nil {
        return fmt.Errorf("min timestamp query failed: %w", err)
    }
    log.Printf("[Cleaner] oldest record is from %s", oldest.Format(time.RFC3339))

    // 3) Compute delete count
    toDelete := totalBefore - maxRecords
    rounded := ((toDelete + deleteMultiple - 1) / deleteMultiple) * deleteMultiple

    // 4) Peek the IDs we’ll delete
    ids := []int{}
    rows, err := db.conn.Query(`
      SELECT id
        FROM builds
       ORDER BY "timestamp" ASC
       LIMIT $1
    `, rounded)
    if err != nil {
        return fmt.Errorf("peek IDs failed: %w", err)
    }
    for rows.Next() {
        var id int
        if err := rows.Scan(&id); err != nil {
            return fmt.Errorf("scan id failed: %w", err)
        }
        ids = append(ids, id)
    }
    rows.Close()

    // 5) Perform the delete
	const deleteSQL = `
	DELETE FROM builds
	WHERE id IN (
		SELECT id FROM builds ORDER BY "timestamp" ASC LIMIT $1
	);
	`
    res, err := db.conn.Exec(deleteSQL, rounded)
    if err != nil {
        return fmt.Errorf("delete failed: %w", err)
    }
    deleted, _ := res.RowsAffected()
    log.Printf("[Cleaner] deleted %d rows", deleted)

    // 6) Total after
    var totalAfter int
    if err := db.conn.QueryRow(`SELECT count(*) FROM builds`).Scan(&totalAfter); err != nil {
        return fmt.Errorf("count after delete failed: %w", err)
    }
    return nil
}

func (db *DB) CountBuilds() (int, error) {
    var total int
    err := db.conn.QueryRow(`SELECT count(*) FROM builds`).Scan(&total)
    return total, err
}
