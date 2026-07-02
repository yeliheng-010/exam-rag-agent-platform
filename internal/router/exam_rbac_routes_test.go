package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

func TestExamStudentRAGRoutesStayViewerAccessible(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enabled := true
	guards := &rbacGuards{
		cfg: &config.Config{Tenant: &config.TenantConfig{EnableRBAC: &enabled}},
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantRoleContextKey, types.TenantRoleViewer)
		ctx = context.WithValue(ctx, types.UserIDContextKey, "student-1")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	router.GET("/knowledge-bases", guards.Viewer(), okHandler)
	router.POST("/knowledge-chat/:session_id", guards.Viewer(), okHandler)
	router.POST("/knowledge-search", guards.Viewer(), okHandler)
	router.POST("/knowledge-bases", guards.Contributor(), okHandler)
	router.POST("/agent-chat/:session_id", guards.Contributor(), okHandler)

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "student can list KBs for chat selection", method: http.MethodGet, path: "/knowledge-bases", want: http.StatusOK},
		{name: "student can ask basic RAG", method: http.MethodPost, path: "/knowledge-chat/s1", want: http.StatusOK},
		{name: "student can use knowledge search", method: http.MethodPost, path: "/knowledge-search", want: http.StatusOK},
		{name: "student cannot create KB", method: http.MethodPost, path: "/knowledge-bases", want: http.StatusForbidden},
		{name: "student cannot use agent chat", method: http.MethodPost, path: "/agent-chat/s1", want: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)
			router.ServeHTTP(recorder, req)
			if recorder.Code != tt.want {
				t.Fatalf("%s %s status = %d, want %d", tt.method, tt.path, recorder.Code, tt.want)
			}
		})
	}
}

func TestExamRAGRouteGuardSourceMatrix(t *testing.T) {
	sourceBytes, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}
	source := string(sourceBytes)

	mustContainAll(t, source, []string{
		`kb.GET("", g.Viewer(), handler.ListKnowledgeBases)`,
		`knowledgeChat := r.Group("/knowledge-chat", g.Viewer())`,
		`knowledgeSearch := r.Group("/knowledge-search", g.Viewer())`,
		`agentChat := r.Group("/agent-chat", g.Contributor())`,
		`kb.POST("", g.Contributor(), handler.CreateKnowledgeBase)`,
	})
}

func TestExamAdminMemberRouteGuardSourceMatrix(t *testing.T) {
	sourceBytes, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}
	source := string(sourceBytes)

	mustContainAll(t, source, []string{
		`tenantByID.POST("/members", g.Admin(), memberHandler.AddMember)`,
		`tenantByID.PUT("/members/:user_id", g.Admin(), memberHandler.UpdateMemberRole)`,
		`tenantByID.DELETE("/members/:user_id", g.Admin(), memberHandler.RemoveMember)`,
		`tenantByID.POST("/invitations", g.Admin(), invitationHandler.CreateInvitation)`,
		`tenantByID.DELETE("/invitations/:inv_id", g.Admin(), invitationHandler.RevokeInvitation)`,
		`tenantByID.POST("/invite-links", g.Admin(), invitationHandler.CreateInviteLink)`,
	})
}

func okHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func mustContainAll(t *testing.T, source string, snippets []string) {
	t.Helper()
	for _, snippet := range snippets {
		if !strings.Contains(source, snippet) {
			t.Fatalf("router guard source missing snippet: %s", snippet)
		}
	}
}
