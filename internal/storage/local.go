package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorageProvider copies completed videos to a local directory.
type LocalStorageProvider struct {
	baseDir string
}

// NewLocalStorageProvider creates a LocalStorageProvider writing to baseDir.
func NewLocalStorageProvider(baseDir string) *LocalStorageProvider {
	if baseDir == "" {
		baseDir = "data/videos"
	}
	return &LocalStorageProvider{
		baseDir: baseDir,
	}
}

func (l *LocalStorageProvider) ProviderName() string {
	return "local"
}

func (l *LocalStorageProvider) Upload(ctx context.Context, localFilePath string, meta UploadMetadata) (*StoredFile, error) {
	if err := os.MkdirAll(l.baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create local storage directory: %w", err)
	}

	srcFile, err := os.Open(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source video file: %w", err)
	}
	defer srcFile.Close()

	srcStat, err := srcFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat source video file: %w", err)
	}
	if srcStat.Size() == 0 {
		return nil, fmt.Errorf("refusing to upload empty video file (0 bytes)")
	}

	destFilename := meta.Filename
	if destFilename == "" {
		destFilename = fmt.Sprintf("%s.mp4", meta.RunID)
	}
	destPath := filepath.Join(l.baseDir, destFilename)

	destFile, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	written, err := io.Copy(destFile, srcFile)
	if err != nil {
		_ = os.Remove(destPath)
		return nil, fmt.Errorf("failed copying video to local storage: %w", err)
	}

	absPath, _ := filepath.Abs(destPath)

	return &StoredFile{
		FileID:      destFilename,
		Filename:    destFilename,
		Size:        written,
		MimeType:    "video/mp4",
		WebViewLink: fmt.Sprintf("file://%s", absPath),
		StorageType: "local",
	}, nil
}
