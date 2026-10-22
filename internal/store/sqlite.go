package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yash-sojitra-20/content-engine/internal/content"
	_ "modernc.org/sqlite"
)

// SQLiteStore persists runs in an embedded SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens or creates an SQLite database at dbPath and runs migrations.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create sqlite directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Set connection pool settings for sqlite single-file write concurrency
	db.SetMaxOpenConns(1)

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to migrate sqlite database: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS runs (
		id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		started_at TIMESTAMP NOT NULL,
		completed_at TIMESTAMP,
		brief_json TEXT,
		provider_job_id TEXT,
		local_video_path TEXT,
		drive_file_id TEXT,
		drive_file_link TEXT,
		error_stage TEXT,
		error_message TEXT,
		retry_count INTEGER DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
	CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs(started_at);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteStore) CreateRun(ctx context.Context, run *Run) error {
	var briefJSON string
	if run.Brief != nil {
		b, err := json.Marshal(run.Brief)
		if err == nil {
			briefJSON = string(b)
		}
	}

	query := `
	INSERT INTO runs (
		id, status, started_at, completed_at, brief_json,
		provider_job_id, local_video_path, drive_file_id,
		drive_file_link, error_stage, error_message, retry_count
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		run.ID,
		string(run.Status),
		run.StartedAt,
		run.CompletedAt,
		briefJSON,
		run.ProviderJobID,
		run.LocalVideoPath,
		run.DriveFileID,
		run.DriveFileLink,
		run.ErrorStage,
		run.ErrorMessage,
		run.RetryCount,
	)
	return err
}

func (s *SQLiteStore) UpdateRun(ctx context.Context, run *Run) error {
	var briefJSON string
	if run.Brief != nil {
		b, err := json.Marshal(run.Brief)
		if err == nil {
			briefJSON = string(b)
		}
	}

	query := `
	UPDATE runs SET
		status = ?,
		completed_at = ?,
		brief_json = ?,
		provider_job_id = ?,
		local_video_path = ?,
		drive_file_id = ?,
		drive_file_link = ?,
		error_stage = ?,
		error_message = ?,
		retry_count = ?
	WHERE id = ?
	`
	res, err := s.db.ExecContext(ctx, query,
		string(run.Status),
		run.CompletedAt,
		briefJSON,
		run.ProviderJobID,
		run.LocalVideoPath,
		run.DriveFileID,
		run.DriveFileLink,
		run.ErrorStage,
		run.ErrorMessage,
		run.RetryCount,
		run.ID,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("run not found")
	}
	return nil
}

func (s *SQLiteStore) GetRun(ctx context.Context, id string) (*Run, error) {
	query := `
	SELECT id, status, started_at, completed_at, brief_json,
	       provider_job_id, local_video_path, drive_file_id,
	       drive_file_link, error_stage, error_message, retry_count
	FROM runs WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var (
		run         Run
		statusStr   string
		completedAt sql.NullTime
		briefJSON   sql.NullString
		pJobID      sql.NullString
		lPath       sql.NullString
		dFileID     sql.NullString
		dFileLink   sql.NullString
		eStage      sql.NullString
		eMsg        sql.NullString
	)

	err := row.Scan(
		&run.ID,
		&statusStr,
		&run.StartedAt,
		&completedAt,
		&briefJSON,
		&pJobID,
		&lPath,
		&dFileID,
		&dFileLink,
		&eStage,
		&eMsg,
		&run.RetryCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("run not found")
		}
		return nil, err
	}

	run.Status = RunStatus(statusStr)
	if completedAt.Valid {
		run.CompletedAt = &completedAt.Time
	}
	if briefJSON.Valid && briefJSON.String != "" {
		var brief content.ContentBrief
		if err := json.Unmarshal([]byte(briefJSON.String), &brief); err == nil {
			run.Brief = &brief
		}
	}
	if pJobID.Valid {
		run.ProviderJobID = pJobID.String
	}
	if lPath.Valid {
		run.LocalVideoPath = lPath.String
	}
	if dFileID.Valid {
		run.DriveFileID = dFileID.String
	}
	if dFileLink.Valid {
		run.DriveFileLink = dFileLink.String
	}
	if eStage.Valid {
		run.ErrorStage = eStage.String
	}
	if eMsg.Valid {
		run.ErrorMessage = eMsg.String
	}

	return &run, nil
}

func (s *SQLiteStore) ListRuns(ctx context.Context, limit int) ([]*Run, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
	SELECT id, status, started_at, completed_at, brief_json,
	       provider_job_id, local_video_path, drive_file_id,
	       drive_file_link, error_stage, error_message, retry_count
	FROM runs
	ORDER BY started_at DESC
	LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Run
	for rows.Next() {
		var (
			run         Run
			statusStr   string
			completedAt sql.NullTime
			briefJSON   sql.NullString
			pJobID      sql.NullString
			lPath       sql.NullString
			dFileID     sql.NullString
			dFileLink   sql.NullString
			eStage      sql.NullString
			eMsg        sql.NullString
		)
		err := rows.Scan(
			&run.ID,
			&statusStr,
			&run.StartedAt,
			&completedAt,
			&briefJSON,
			&pJobID,
			&lPath,
			&dFileID,
			&dFileLink,
			&eStage,
			&eMsg,
			&run.RetryCount,
		)
		if err != nil {
			return nil, err
		}

		run.Status = RunStatus(statusStr)
		if completedAt.Valid {
			run.CompletedAt = &completedAt.Time
		}
		if briefJSON.Valid && briefJSON.String != "" {
			var brief content.ContentBrief
			if err := json.Unmarshal([]byte(briefJSON.String), &brief); err == nil {
				run.Brief = &brief
			}
		}
		if pJobID.Valid {
			run.ProviderJobID = pJobID.String
		}
		if lPath.Valid {
			run.LocalVideoPath = lPath.String
		}
		if dFileID.Valid {
			run.DriveFileID = dFileID.String
		}
		if dFileLink.Valid {
			run.DriveFileLink = dFileLink.String
		}
		if eStage.Valid {
			run.ErrorStage = eStage.String
		}
		if eMsg.Valid {
			run.ErrorMessage = eMsg.String
		}

		list = append(list, &run)
	}

	return list, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
