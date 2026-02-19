package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
	path string
}

func New(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	wrapper := &DB{DB: db, path: dbPath}
	if err := wrapper.Migrate(); err != nil {
		return nil, err
	}

	if err := os.Chmod(dbPath, 0600); err != nil {
		return nil, err
	}

	return wrapper, nil
}

func (db *DB) Migrate() error {
	if _, err := os.Stat(db.path); err == nil {
		if err := os.Chmod(db.path, 0600); err != nil {
			return err
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
CREATE TABLE IF NOT EXISTS usage_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    collected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    subscription_limit INTEGER NOT NULL,
    requests_used INTEGER NOT NULL,
    leftover INTEGER GENERATED ALWAYS AS (subscription_limit - requests_used) STORED,
    renews_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_collected_at ON usage_snapshots(collected_at);

CREATE VIEW IF NOT EXISTS daily_usage AS
SELECT 
    DATE(collected_at) as day,
    MAX(requests_used) - MIN(requests_used) as requests_consumed,
    MIN(leftover) as min_leftover,
    MAX(leftover) as max_leftover,
    AVG(leftover) as avg_leftover,
    COUNT(*) as snapshots
FROM usage_snapshots
GROUP BY DATE(collected_at)
ORDER BY day DESC;

CREATE VIEW IF NOT EXISTS weekly_usage AS
SELECT 
    strftime('%Y-W%W', collected_at) as week,
    MAX(requests_used) - MIN(requests_used) as requests_consumed,
    MIN(leftover) as min_leftover,
    MAX(leftover) as max_leftover,
    AVG(leftover) as avg_leftover,
    COUNT(*) as snapshots
FROM usage_snapshots
GROUP BY strftime('%Y-W%W', collected_at)
ORDER BY week DESC;
	`)
	if err != nil {
		return err
	}

	return tx.Commit()
}
