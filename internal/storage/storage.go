package storage

import (
	"context"
	"time"
)

// UploadMetadata contains descriptive metadata for the video file being stored.
type UploadMetadata struct {
	RunID       string
	Title       string
	Filename    string
	ContentType string
	CreatedAt   time.Time
}

// StoredFile represents the result of a successful upload.
type StoredFile struct {
	FileID      string `json:"file_id"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	MimeType    string `json:"mime_type"`
	WebViewLink string `json:"web_view_link"`
	StorageType string `json:"storage_type"`
}

// StorageProvider defines the contract for persisting generated video files.
type StorageProvider interface {
	ProviderName() string
	Upload(ctx context.Context, localFilePath string, meta UploadMetadata) (*StoredFile, error)
}
