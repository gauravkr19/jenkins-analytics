package models

import "time"

type Build struct {
	ID             int       `db:"id"`
	BuildNumber    int       `db:"build_number"`
	ProjectName    string    `db:"project_name"`
	UserID         string    `db:"user_id"`
	Status         string    `db:"status"`
	Result         string    `db:"result"`
	Timestamp      time.Time `db:"timestamp"`
	DurationMS     int64     `db:"duration_ms"`
	Branch         string    `db:"branch"`
	CommitID       string    `db:"commit_id"`
	JobURL         string    `db:"job_url"`
	ConsoleLogHead string    `db:"console_log_head"`
	ConsoleLogTail string    `db:"console_log_tail"`
	ErrorMessage   string    `db:"error_message"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}
