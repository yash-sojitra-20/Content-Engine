package store

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// JSONLStore persists runs in an append-oriented JSONL file.
type JSONLStore struct {
	mu       sync.RWMutex
	filePath string
}

// NewJSONLStore creates or opens a JSONL store at the specified path.
func NewJSONLStore(filePath string) (*JSONLStore, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory for jsonl store: %w", err)
	}

	// Touch file if not exists
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open jsonl file: %w", err)
	}
	_ = f.Close()

	return &JSONLStore{filePath: filePath}, nil
}

func (j *JSONLStore) CreateRun(ctx context.Context, run *Run) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	data, err := json.Marshal(run)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(j.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (j *JSONLStore) UpdateRun(ctx context.Context, run *Run) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	runs, err := j.readAllUnlocked()
	if err != nil {
		return err
	}

	found := false
	for i, r := range runs {
		if r.ID == run.ID {
			runs[i] = run
			found = true
			break
		}
	}

	if !found {
		// If not found, append
		runs = append(runs, run)
	}

	return j.rewriteAllUnlocked(runs)
}

func (j *JSONLStore) GetRun(ctx context.Context, id string) (*Run, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	runs, err := j.readAllUnlocked()
	if err != nil {
		return nil, err
	}

	for _, r := range runs {
		if r.ID == id {
			return r, nil
		}
	}

	return nil, errors.New("run not found")
}

func (j *JSONLStore) ListRuns(ctx context.Context, limit int) ([]*Run, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	runs, err := j.readAllUnlocked()
	if err != nil {
		return nil, err
	}

	if limit <= 0 || limit >= len(runs) {
		return runs, nil
	}

	// Return most recent runs
	return runs[len(runs)-limit:], nil
}

func (j *JSONLStore) readAllUnlocked() ([]*Run, error) {
	f, err := os.Open(j.filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var runs []*Run
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r Run
		if err := json.Unmarshal(line, &r); err == nil {
			runs = append(runs, &r)
		}
	}
	return runs, scanner.Err()
}

func (j *JSONLStore) rewriteAllUnlocked(runs []*Run) error {
	tmpPath := j.filePath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	for _, r := range runs {
		data, err := json.Marshal(r)
		if err != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
			return err
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
			return err
		}
	}

	if err := f.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, j.filePath)
}

func (j *JSONLStore) Close() error {
	return nil
}
