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
	JobURL      string    `db:"job_url"`
	Branch      string    `db:"branch"`
	GitRepo     string    `db:"git_url"`
	CommitSHA   string    `db:"commit_sha"`
	DeployEnv   string    `db:"deploy_env"`   // params
	TriggerType string    `db:"trigger_type"` // cause.shortDescription
<<<<<<< Updated upstream
<<<<<<< Updated upstream
=======
=======
>>>>>>> Stashed changes
}

// models/folder_tree.go
type FolderNode struct {
	Name     string
	FullPath string
	IsLeaf   bool
	Children map[string]*FolderNode
<<<<<<< Updated upstream
>>>>>>> Stashed changes
=======
>>>>>>> Stashed changes
}

type BuildLog struct {
	ID             int    `db:"id"`
	BuildNumber    int    `db:"build_number"`
	ProjectName    string `db:"project_name"`
	ConsoleLogHead string `db:"console_log_head"`
	ConsoleLogTail string `db:"console_log_tail"`
}
