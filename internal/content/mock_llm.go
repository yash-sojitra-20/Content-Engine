package content

import (
	"context"
	"fmt"
)

// MockLLMClient implements LLMProvider for unit testing and offline development.
type MockLLMClient struct {
	CustomBrief *ContentBrief
	SimulateErr error
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{}
}

func (m *MockLLMClient) ProviderName() string {
	return "mock"
}

func (m *MockLLMClient) GenerateContentBrief(ctx context.Context, topicPrompt string) (*ContentBrief, error) {
	if m.SimulateErr != nil {
		return nil, m.SimulateErr
	}

	if m.CustomBrief != nil {
		return m.CustomBrief, nil
	}

	brief := &ContentBrief{
		Title:           "How Generative AI Actually Works",
		Topic:           fmt.Sprintf("Tech Explainer: %s", topicPrompt),
		TargetAudience:  "Developers & tech enthusiasts",
		Script:          "Generative AI predicts the next most likely token. In 10 seconds, here is the secret of neural weights.",
		DurationSeconds: 10,
		AspectRatio:     "9:16",
		VisualStyle:     "Futuristic 3D data visualization, cinematic neon blue glow",
		GenerationPrompt: "Cinematic 3D animation of glowing neural network pathways pulsing with electric blue energy, seamlessly connecting data nodes in a dark void, 4k hyperrealistic, smooth camera zoom-in.",
		Scenes: []Scene{
			{
				SceneNumber: 1,
				Description: "Camera rushes through glowing neural synapses",
				VisualCue:   "Neon blue electric pulses",
				Narration:   "Generative AI predicts the next most likely token.",
			},
			{
				SceneNumber: 2,
				Description: "A glowing 3D cube emerges with floating mathematical vectors",
				VisualCue:   "Holographic grid",
				Narration:   "Neural weights transform vectors into living intelligence.",
			},
		},
	}

	return brief, nil
}
