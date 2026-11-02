package social

import (
	"context"
	"time"
)

// PublishRequest contains all parameters necessary to publish a video to a social platform.
type PublishRequest struct {
	// RunID is the unique identifier of the content-engine run.
	RunID string

	// FilePath is the absolute or relative path to the local video file.
	FilePath string

	// Title is the video title.
	Title string

	// Description is the description / caption for the post.
	Description string

	// Tags is a list of topic or hashtag keywords.
	Tags []string

	// PrivacyStatus determines visibility: "public", "unlisted", or "private".
	PrivacyStatus string

	// Category is the platform-specific category ID (e.g. YouTube category "28" for Tech).
	Category string

	// IsShort specifies whether this video is formatted as a vertical short/reel (e.g. 9:16 aspect ratio).
	IsShort bool
}

// PublishResult contains metadata returned by the social media platform after successful publication.
type PublishResult struct {
	// Platform identifies the social network (e.g. "youtube", "mock").
	Platform string

	// PostID is the unique identifier assigned to the uploaded video by the platform.
	PostID string

	// PostURL is the direct public/viewable URL to the published post or short.
	PostURL string

	// PublishedAt is the timestamp when the publication completed.
	PublishedAt time.Time

	// Metadata holds any extra platform-specific key-value details.
	Metadata map[string]string
}

// Publisher defines the contract for uploading and publishing video content to social platforms.
type Publisher interface {
	// Publish uploads the video to the target platform and returns the resulting post information.
	Publish(ctx context.Context, req PublishRequest) (*PublishResult, error)

	// PlatformName returns the identifier for this publishing provider (e.g. "youtube", "mock").
	PlatformName() string
}
