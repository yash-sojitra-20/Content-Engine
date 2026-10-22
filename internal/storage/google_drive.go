package storage

import (
	"context"
	"fmt"
	"os"

	"github.com/yash-sojitra-20/content-engine/internal/retry"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GoogleDriveProvider uploads files to a Google Drive folder using the Drive v3 API.
type GoogleDriveProvider struct {
	folderID string
	service  *drive.Service
}

// NewGoogleDriveProvider initializes a Google Drive client with the given credentials file.
func NewGoogleDriveProvider(ctx context.Context, credentialsFilePath, folderID string) (*GoogleDriveProvider, error) {
	if credentialsFilePath == "" {
		return nil, retry.MarkNonRetryable(fmt.Errorf("credentials file path is required"))
	}
	if folderID == "" {
		return nil, retry.MarkNonRetryable(fmt.Errorf("destination folder ID is required"))
	}

	if _, err := os.Stat(credentialsFilePath); err != nil {
		return nil, retry.MarkNonRetryable(fmt.Errorf("google credentials file not found at %s: %w", credentialsFilePath, err))
	}

	srv, err := drive.NewService(ctx,
		option.WithCredentialsFile(credentialsFilePath),
		option.WithScopes(drive.DriveFileScope),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize google drive service: %w", err)
	}

	return &GoogleDriveProvider{
		folderID: folderID,
		service:  srv,
	}, nil
}

func (g *GoogleDriveProvider) ProviderName() string {
	return "gdrive"
}

func (g *GoogleDriveProvider) Upload(ctx context.Context, localFilePath string, meta UploadMetadata) (*StoredFile, error) {
	file, err := os.Open(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local video file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat video file: %w", err)
	}
	if stat.Size() == 0 {
		return nil, retry.MarkNonRetryable(fmt.Errorf("refusing to upload empty video file (0 bytes)"))
	}

	filename := meta.Filename
	if filename == "" {
		filename = fmt.Sprintf("content_engine_%s.mp4", meta.RunID)
	}

	driveFile := &drive.File{
		Name:        filename,
		Parents:     []string{g.folderID},
		MimeType:    "video/mp4",
		Description: fmt.Sprintf("Run ID: %s | Title: %s", meta.RunID, meta.Title),
	}

	call := g.service.Files.Create(driveFile).
		SupportsAllDrives(true).
		Media(file).
		Fields("id, name, mimeType, size, webViewLink").
		Context(ctx)

	res, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("google drive upload failed: %w", err)
	}

	return &StoredFile{
		FileID:      res.Id,
		Filename:    res.Name,
		Size:        res.Size,
		MimeType:    res.MimeType,
		WebViewLink: res.WebViewLink,
		StorageType: "gdrive",
	}, nil
}
