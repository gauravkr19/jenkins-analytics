package models

import "time"

type Build struct {
	ID          int       `db:"id"`
	BuildNumber int       `db:"build_number"`
	ProjectName string    `db:"project_name"`
	ProjectPath string    `db:"project_path"`
	UserID      string    `db:"user_id"`
	Status      string    `db:"status"`
	Timestamp   time.Time `db:"timestamp"`
	DurationMS  int64     `db:"duration_ms"`
	Branch      string    `db:"branch"`
	JobURL      string    `db:"job_url"`
}

type BuildLog struct {
	ID             int    `db:"id"`
	BuildNumber    int    `db:"build_number"`
	ProjectName    string `db:"project_name"`
	ConsoleLogHead string `db:"console_log_head"`
	ConsoleLogTail string `db:"console_log_tail"`
}
