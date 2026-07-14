package router

import (
	"os"
	"strings"
	"testing"
)

func TestExamQuestionGroupExtractionRegisteredInBothTaskExecutors(t *testing.T) {
	for _, file := range []string{"task.go", "sync_task.go"} {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if !strings.Contains(string(source), "types.TypeExamQuestionGroupExtraction") {
			t.Fatalf("%s does not register exam question group extraction", file)
		}
		if !strings.Contains(string(source), "QuestionGroupDraftService.ProcessExtractionTask") {
			t.Fatalf("%s does not use the shared extraction task handler", file)
		}
	}
}
