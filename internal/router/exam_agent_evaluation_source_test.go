package router

import (
	"os"
	"strings"
	"testing"
)

func TestExamAgentEvaluationRoutesRequireContributor(t *testing.T) {
	source, err := os.ReadFile("exam.go")
	if err != nil {
		t.Fatalf("read exam.go: %v", err)
	}
	for _, snippet := range []string{
		`exam.POST("/question-banks/:bank_id/agent-evaluation-runs", g.Contributor(), agentEvaluationHandler.CreateRun)`,
		`exam.GET("/question-banks/:bank_id/agent-evaluation-runs", g.Contributor(), agentEvaluationHandler.ListRuns)`,
		`exam.GET("/question-banks/:bank_id/agent-evaluation-runs/:run_id", g.Contributor(), agentEvaluationHandler.GetRun)`,
	} {
		if !strings.Contains(string(source), snippet) {
			t.Fatalf("exam routes missing contributor endpoint: %s", snippet)
		}
	}
}

func TestExamAgentEvaluationTaskRegisteredInBothExecutors(t *testing.T) {
	for _, file := range []string{"task.go", "sync_task.go"} {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		content := string(source)
		if !strings.Contains(content, "types.TypeExamAgentEvaluationRun") ||
			!strings.Contains(content, "AgentEvaluationService.ProcessRunTask") {
			t.Fatalf("%s does not register the Agent evaluation worker", file)
		}
	}
}
