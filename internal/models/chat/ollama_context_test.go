package chat

import "testing"

func TestOllamaChatReservesContextForPromptAndCompletion(t *testing.T) {
	tests := []struct {
		name      string
		maxTokens int
		want      int
	}{
		{name: "minimum window", maxTokens: 2048, want: 16384},
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

func TestOllamaChatUsesMaxCompletionTokens(t *testing.T) {
	req := (&OllamaChat{}).buildChatRequest(nil, &ChatOptions{MaxCompletionTokens: 2048}, false)

	if got := req.Options["num_predict"]; got != 2048 {
		t.Fatalf("expected num_predict 2048, got %v", got)
	}
	if got := req.Options["num_ctx"]; got != 16384 {
		t.Fatalf("expected num_ctx 16384, got %v", got)
	}
}
