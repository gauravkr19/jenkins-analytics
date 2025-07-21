-- 1. Create builds table
CREATE TABLE IF NOT EXISTS builds (
    id SERIAL PRIMARY KEY,
    build_number INT NOT NULL,
    project_name TEXT,
    project_path TEXT NOT NULL,
    status TEXT,
    timestamp TIMESTAMP,
    duration_ms BIGINT,
    job_url TEXT,
    user_id TEXT,
    git_url TEXT,
    branch TEXT,
    commit_sha TEXT,
    deploy_env TEXT,
    trigger_type TEXT
);

-- 2. Ensure uniqueness on build_number + project_path to avoid duplicates
DO \$\$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'unique_build_path'
    ) THEN
        ALTER TABLE builds
        ADD CONSTRAINT unique_build_path UNIQUE (build_number, project_path);
    END IF;
END\$\$;


-- 3. Create sync_status table to track full sync state
CREATE TABLE IF NOT EXISTS sync_status (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at TIMESTAMPTZ DEFAULT now()
);
