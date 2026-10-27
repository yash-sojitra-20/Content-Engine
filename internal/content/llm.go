package content

import (
	"context"
)

// LLMProvider defines the contract for generating structured video content briefs.
type LLMProvider interface {
	// GenerateContentBrief calls the underlying model to produce a validated ContentBrief.
	GenerateContentBrief(ctx context.Context, topicPrompt string) (*ContentBrief, error)
	// ProviderName returns the identifier of the LLM provider.
	ProviderName() string
}
