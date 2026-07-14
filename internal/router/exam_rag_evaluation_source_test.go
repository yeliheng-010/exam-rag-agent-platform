package router

import (
	"os"
	"strings"
	"testing"
)

func TestExamRAGEvaluationRoutesRequireContributor(t *testing.T) {
	source, err := os.ReadFile("exam.go")
	if err != nil {
		t.Fatalf("read exam.go: %v", err)
	}
	for _, snippet := range []string{
		`exam.POST("/question-banks/:bank_id/rag-evaluation-runs", g.Contributor(), ragEvaluationHandler.CreateRun)`,
		`exam.GET("/question-banks/:bank_id/rag-evaluation-runs", g.Contributor(), ragEvaluationHandler.ListRuns)`,
		`exam.GET("/question-banks/:bank_id/rag-evaluation-runs/:run_id", g.Contributor(), ragEvaluationHandler.GetRun)`,
	} {
		if !strings.Contains(string(source), snippet) {
			t.Fatalf("exam routes missing contributor endpoint: %s", snippet)
		}
	}
}

func TestExamRAGEvaluationTaskRegisteredInBothExecutors(t *testing.T) {
	for _, file := range []string{"task.go", "sync_task.go"} {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		content := string(source)
		if !strings.Contains(content, "types.TypeExamRAGEvaluationRun") ||
			!strings.Contains(content, "RAGEvaluationService.ProcessRunTask") {
			t.Fatalf("%s does not register the shared RAG evaluation worker", file)
		}
	}
}
