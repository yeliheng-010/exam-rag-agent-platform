package chat

import "testing"

func TestOllamaChatReservesContextForPromptAndCompletion(t *testing.T) {
	tests := []struct {
		name      string
		maxTokens int
		want      int
	}{
		{name: "minimum window", maxTokens: 2048, want: 8192},
		{name: "large completion", maxTokens: 8192, want: 16384},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := (&OllamaChat{}).buildChatRequest(nil, &ChatOptions{MaxTokens: tt.maxTokens}, false)
			if got := req.Options["num_ctx"]; got != tt.want {
				t.Fatalf("expected num_ctx %d, got %v", tt.want, got)
			}
		})
	}
}
