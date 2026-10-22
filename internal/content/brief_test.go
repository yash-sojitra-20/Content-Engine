package content

import (
	"context"
	"testing"
)

func TestContentBriefValidation(t *testing.T) {
	tests := []struct {
		name      string
		brief     ContentBrief
		wantError bool
	}{
		{
			name: "Valid brief",
			brief: ContentBrief{
				Title:            "Micro-fact on Compilers",
				Topic:            "Tech",
				GenerationPrompt: "Camera moves over silicon chip with laser lines",
				DurationSeconds:  10,
				AspectRatio:      "9:16",
			},
			wantError: false,
		},
		{
			name: "Missing title",
			brief: ContentBrief{
				Topic:            "Tech",
				GenerationPrompt: "Prompt",
				DurationSeconds:  10,
				AspectRatio:      "9:16",
			},
			wantError: true,
		},
		{
			name: "Invalid duration",
			brief: ContentBrief{
				Title:            "Title",
				Topic:            "Tech",
				GenerationPrompt: "Prompt",
				DurationSeconds:  0,
				AspectRatio:      "9:16",
			},
			wantError: true,
		},
		{
			name: "Invalid aspect ratio",
			brief: ContentBrief{
				Title:            "Title",
				Topic:            "Tech",
				GenerationPrompt: "Prompt",
				DurationSeconds:  10,
				AspectRatio:      "4:3",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.brief.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCleanJSONMarkdown(t *testing.T) {
	input := "```json\n{\"title\": \"test\"}\n```"
	cleaned := cleanJSONMarkdown(input)
	if cleaned != "{\"title\": \"test\"}" {
		t.Errorf("unexpected cleaned result: %q", cleaned)
	}
}

func TestMockLLMClient(t *testing.T) {
	mock := NewMockLLMClient()
	brief, err := mock.GenerateContentBrief(context.Background(), "AI Agents")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if brief.Title == "" || brief.GenerationPrompt == "" {
		t.Errorf("invalid brief generated: %+v", brief)
	}
	if err := brief.Validate(); err != nil {
		t.Errorf("mock brief should be valid: %v", err)
	}
}
