package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/yash-sojitra-20/content-engine/internal/content"
)

// PostgresStore persists runs in a PostgreSQL database.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore connects to PostgreSQL using the provided connection string and runs migrations.
func NewPostgresStore(connStr string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	store := &PostgresStore{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to migrate postgres database: %w", err)
	}

	return store, nil
}

func (p *PostgresStore) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS runs (
		id VARCHAR(100) PRIMARY KEY,
		status VARCHAR(50) NOT NULL,
		started_at TIMESTAMPTZ NOT NULL,
		completed_at TIMESTAMPTZ,
		brief_json TEXT,
		provider_job_id VARCHAR(255),
		local_video_path TEXT,
		drive_file_id VARCHAR(255),
		drive_file_link TEXT,
		social_platform VARCHAR(100),
		social_post_id VARCHAR(255),
		social_post_url TEXT,
		error_stage VARCHAR(100),
		error_message TEXT,
		retry_count INT DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
	CREATE INDEX IF NOT EXISTS idx_runs_started_at ON runs(started_at);
	`
	if _, err := p.db.Exec(query); err != nil {
		return err
	}

	// Backwards-compatible column additions
	_, _ = p.db.Exec("ALTER TABLE runs ADD COLUMN IF NOT EXISTS social_platform VARCHAR(100);")
	_, _ = p.db.Exec("ALTER TABLE runs ADD COLUMN IF NOT EXISTS social_post_id VARCHAR(255);")
	_, _ = p.db.Exec("ALTER TABLE runs ADD COLUMN IF NOT EXISTS social_post_url TEXT;")

	return nil
}

func (p *PostgresStore) CreateRun(ctx context.Context, run *Run) error {
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
		drive_file_link, social_platform, social_post_id, social_post_url,
		error_stage, error_message, retry_count
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := p.db.ExecContext(ctx, query,
		run.ID,
		string(run.Status),
		run.StartedAt,
		run.CompletedAt,
		briefJSON,
		run.ProviderJobID,
		run.LocalVideoPath,
		run.DriveFileID,
		run.DriveFileLink,
		run.SocialPlatform,
		run.SocialPostID,
		run.SocialPostURL,
		run.ErrorStage,
		run.ErrorMessage,
		run.RetryCount,
	)
	return err
}

func (p *PostgresStore) UpdateRun(ctx context.Context, run *Run) error {
	var briefJSON string
	if run.Brief != nil {
		b, err := json.Marshal(run.Brief)
		if err == nil {
			briefJSON = string(b)
		}
	}

	query := `
	UPDATE runs SET
		status = $1,
		completed_at = $2,
		brief_json = $3,
		provider_job_id = $4,
		local_video_path = $5,
		drive_file_id = $6,
		drive_file_link = $7,
		social_platform = $8,
		social_post_id = $9,
		social_post_url = $10,
		error_stage = $11,
		error_message = $12,
		retry_count = $13
	WHERE id = $14
	`
	res, err := p.db.ExecContext(ctx, query,
		string(run.Status),
		run.CompletedAt,
		briefJSON,
		run.ProviderJobID,
		run.LocalVideoPath,
		run.DriveFileID,
		run.DriveFileLink,
		run.SocialPlatform,
		run.SocialPostID,
		run.SocialPostURL,
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

func (p *PostgresStore) GetRun(ctx context.Context, id string) (*Run, error) {
	query := `
	SELECT id, status, started_at, completed_at, brief_json,
	       provider_job_id, local_video_path, drive_file_id,
	       drive_file_link, social_platform, social_post_id, social_post_url,
	       error_stage, error_message, retry_count
	FROM runs WHERE id = $1
	`
	row := p.db.QueryRowContext(ctx, query, id)

	var (
		run         Run
		statusStr   string
		completedAt sql.NullTime
		briefJSON   sql.NullString
		pJobID      sql.NullString
		lPath       sql.NullString
		dFileID     sql.NullString
		dFileLink   sql.NullString
		sPlatform   sql.NullString
		sPostID     sql.NullString
		sPostURL    sql.NullString
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
		&sPlatform,
		&sPostID,
		&sPostURL,
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
	if sPlatform.Valid {
		run.SocialPlatform = sPlatform.String
	}
	if sPostID.Valid {
		run.SocialPostID = sPostID.String
	}
	if sPostURL.Valid {
		run.SocialPostURL = sPostURL.String
	}
	if eStage.Valid {
		run.ErrorStage = eStage.String
	}
	if eMsg.Valid {
		run.ErrorMessage = eMsg.String
	}

	return &run, nil
}

func (p *PostgresStore) ListRuns(ctx context.Context, limit int) ([]*Run, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `
	SELECT id, status, started_at, completed_at, brief_json,
	       provider_job_id, local_video_path, drive_file_id,
	       drive_file_link, social_platform, social_post_id, social_post_url,
	       error_stage, error_message, retry_count
	FROM runs
	ORDER BY started_at DESC
	LIMIT $1
	`
	rows, err := p.db.QueryContext(ctx, query, limit)
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
			sPlatform   sql.NullString
			sPostID     sql.NullString
			sPostURL    sql.NullString
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
			&sPlatform,
			&sPostID,
			&sPostURL,
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
		if sPlatform.Valid {
			run.SocialPlatform = sPlatform.String
		}
		if sPostID.Valid {
			run.SocialPostID = sPostID.String
		}
		if sPostURL.Valid {
			run.SocialPostURL = sPostURL.String
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

func (p *PostgresStore) Close() error {
	return p.db.Close()
}
