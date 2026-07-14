package handler

import (
	"os"
	"strings"
	"testing"
)

func TestQuestionGroupExtractionHTTPStartsBackgroundTask(t *testing.T) {
	source, err := os.ReadFile("exam_question_group_draft.go")
	if err != nil {
		t.Fatalf("read handler source: %v", err)
	}
	text := string(source)
	if !strings.Contains(text, "h.service.StartExtraction") {
		t.Fatal("group extraction handler must start the background task")
	}
	if !strings.Contains(text, "http.StatusAccepted") {
		t.Fatal("group extraction handler must return HTTP 202")
	}
}
