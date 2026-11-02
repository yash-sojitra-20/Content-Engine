package social

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yash-sojitra-20/content-engine/internal/retry"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// YouTubeConfig contains connection and default publication options for the YouTube Data API v3.
type YouTubeConfig struct {
	// OAuth2 Client Credentials
	ClientID     string
	ClientSecret string
	RefreshToken string
	AccessToken  string

	// File path to Google OAuth client_secret.json or service account credentials
	CredentialsFile string

	// Default publication settings
	PrivacyStatus   string // "public", "unlisted", "private"
	DefaultCategory string // e.g. "28" for Science & Technology
}

// YouTubePublisher uploads videos and shorts to YouTube via the official YouTube Data API v3.
type YouTubePublisher struct {
	cfg     YouTubeConfig
	service *youtube.Service
}

// NewYouTubePublisher initializes a YouTube Data API client using provided credentials.
func NewYouTubePublisher(ctx context.Context, cfg YouTubeConfig) (*YouTubePublisher, error) {
	var opts []option.ClientOption

	switch {
	case cfg.AccessToken != "":
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.AccessToken})
		opts = append(opts, option.WithTokenSource(tokenSource))

	case cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.RefreshToken != "":
		oauthConf := &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       []string{youtube.YoutubeUploadScope},
		}
		token := &oauth2.Token{
			RefreshToken: cfg.RefreshToken,
		}
		opts = append(opts, option.WithTokenSource(oauthConf.TokenSource(ctx, token)))

	case cfg.CredentialsFile != "":
		if _, err := os.Stat(cfg.CredentialsFile); err != nil {
			return nil, retry.MarkNonRetryable(fmt.Errorf("youtube credentials file not found at %s: %w", cfg.CredentialsFile, err))
		}
		opts = append(opts,
			option.WithCredentialsFile(cfg.CredentialsFile),
			option.WithScopes(youtube.YoutubeUploadScope),
		)

	default:
		return nil, retry.MarkNonRetryable(fmt.Errorf("no valid YouTube credentials provided: require either (CLIENT_ID, CLIENT_SECRET, REFRESH_TOKEN) or CREDENTIALS_FILE"))
	}

	srv, err := youtube.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize youtube service: %w", err)
	}

	return NewYouTubePublisherWithService(srv, cfg), nil
}

// NewYouTubePublisherWithService creates a publisher using an existing YouTube API service instance.
func NewYouTubePublisherWithService(srv *youtube.Service, cfg YouTubeConfig) *YouTubePublisher {
	if cfg.PrivacyStatus == "" {
		cfg.PrivacyStatus = "public"
	}
	if cfg.DefaultCategory == "" {
		cfg.DefaultCategory = "28" // Science & Technology
	}
	return &YouTubePublisher{
		cfg:     cfg,
		service: srv,
	}
}

func (y *YouTubePublisher) PlatformName() string {
	return "youtube"
}

// Publish uploads the video to YouTube and configures Shorts metadata.
func (y *YouTubePublisher) Publish(ctx context.Context, req PublishRequest) (*PublishResult, error) {
	if req.FilePath == "" {
		return nil, retry.MarkNonRetryable(fmt.Errorf("video file path is required"))
	}

	file, err := os.Open(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat video file: %w", err)
	}
	if stat.Size() == 0 {
		return nil, retry.MarkNonRetryable(fmt.Errorf("refusing to upload empty video file (0 bytes)"))
	}

	title := formatShortsTitle(req.Title, req.IsShort)

	desc := req.Description
	if req.IsShort && !strings.Contains(strings.ToLower(desc), "#shorts") {
		desc = desc + "\n\n#Shorts"
	}

	privacy := req.PrivacyStatus
	if privacy == "" {
		privacy = y.cfg.PrivacyStatus
	}

	category := req.Category
	if category == "" {
		category = y.cfg.DefaultCategory
	}

	video := &youtube.Video{
		Snippet: &youtube.VideoSnippet{
			Title:       title,
			Description: desc,
			Tags:        req.Tags,
			CategoryId:  category,
		},
		Status: &youtube.VideoStatus{
			PrivacyStatus:           privacy,
			SelfDeclaredMadeForKids: false,
		},
	}

	call := y.service.Videos.Insert([]string{"snippet", "status"}, video).
		Media(file).
		Context(ctx)

	res, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("youtube video upload failed: %w", err)
	}

	postURL := fmt.Sprintf("https://youtube.com/shorts/%s", res.Id)
	if !req.IsShort {
		postURL = fmt.Sprintf("https://youtu.be/%s", res.Id)
	}

	return &PublishResult{
		Platform:    "youtube",
		PostID:      res.Id,
		PostURL:     postURL,
		PublishedAt: time.Now().UTC(),
		Metadata: map[string]string{
			"title":          title,
			"privacy_status": privacy,
			"category_id":    category,
			"is_short":       fmt.Sprintf("%t", req.IsShort),
		},
	}, nil
}

// formatShortsTitle ensures YouTube Shorts are tagged with #Shorts and respect YouTube's 100 character title limit.
func formatShortsTitle(title string, isShort bool) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Autonomous Video Short"
	}

	if isShort && !strings.Contains(strings.ToLower(title), "#shorts") {
		if len(title)+8 <= 100 {
			title = title + " #Shorts"
		}
	}

	if len(title) > 100 {
		title = strings.TrimSpace(title[:97]) + "..."
	}

	return title
}
