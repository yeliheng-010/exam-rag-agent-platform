package router

import (
	"os"
	"testing"
)

func TestExamClassRouteGuardSourceMatrix(t *testing.T) {
	sourceBytes, err := os.ReadFile("exam.go")
	if err != nil {
		t.Fatalf("read exam.go: %v", err)
	}
	mustContainAll(t, string(sourceBytes), []string{
		`exam.GET("/classes", g.Viewer(), classHandler.ListClasses)`,
		`exam.PUT("/classes/:class_id", g.Viewer(), classHandler.UpdateClass)`,
		`exam.POST("/classes/:class_id/archive", g.Viewer(), classHandler.ArchiveClass)`,
		`exam.POST("/classes/:class_id/restore", g.Viewer(), classHandler.RestoreClass)`,
	})
}
