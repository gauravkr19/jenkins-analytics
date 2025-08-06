-- Init script does not initialize when it exists in /docker-entrypoint-initdb.d
-- Create user if needed
CREATE USER jenkins WITH PASSWORD 'jenkins';

-- Connect to correct database
\connect jenkins

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
    trigger_type TEXT,
    env TEXT
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
END \$\$;


-- 3. Create sync_status table to track full sync state
CREATE TABLE IF NOT EXISTS sync_status (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- Permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO jenkins;
GRANT USAGE, SELECT, UPDATE ON SEQUENCE builds_id_seq TO jenkins;
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO jenkins;
