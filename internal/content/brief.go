package content

import (
	"errors"
	"fmt"
	"strings"
)

// Scene represents a single scene or visual beat in a content brief.
type Scene struct {
	SceneNumber int    `json:"scene_number"`
	Description string `json:"description"`
	VisualCue   string `json:"visual_cue"`
	Narration   string `json:"narration,omitempty"`
}

// ContentBrief holds the structured creative and technical specification for generating a video.
type ContentBrief struct {
	Title            string  `json:"title"`
	Topic            string  `json:"topic"`
	TargetAudience   string  `json:"target_audience"`
	Script           string  `json:"script"`
	Scenes           []Scene `json:"scenes,omitempty"`
	DurationSeconds  int     `json:"duration_seconds"`
	AspectRatio      string  `json:"aspect_ratio"` // e.g. "9:16", "16:9", "1:1"
	VisualStyle      string  `json:"visual_style"`
	GenerationPrompt string  `json:"generation_prompt"`
}

// Validate checks whether the brief contains all necessary non-empty fields and valid values.
func (b *ContentBrief) Validate() error {
	var errs []string

	if strings.TrimSpace(b.Title) == "" {
		errs = append(errs, "title cannot be empty")
	}
	if strings.TrimSpace(b.Topic) == "" {
		errs = append(errs, "topic cannot be empty")
	}
	if strings.TrimSpace(b.GenerationPrompt) == "" {
		errs = append(errs, "generation_prompt cannot be empty")
	}

	if b.DurationSeconds <= 0 || b.DurationSeconds > 300 {
		errs = append(errs, fmt.Sprintf("duration_seconds must be between 1 and 300, got %d", b.DurationSeconds))
	}

	// Normalize Aspect Ratio
	ar := strings.TrimSpace(b.AspectRatio)
	if ar == "" {
		b.AspectRatio = "9:16" // Default vertical video for reels/shorts
	} else if ar != "9:16" && ar != "16:9" && ar != "1:1" {
		errs = append(errs, fmt.Sprintf("unsupported aspect_ratio %q (allowed: '9:16', '16:9', '1:1')", ar))
	}

	if len(errs) > 0 {
		return errors.New("invalid content brief: " + strings.Join(errs, "; "))
	}
	return nil
}
