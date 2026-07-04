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
	router.GET("/exam/resources", guards.Viewer(), okHandler)
	router.POST("/exam/resources/knowledge-bases/:kb_id/bind", guards.Contributor(), okHandler)

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "student can list KBs for chat selection", method: http.MethodGet, path: "/knowledge-bases", want: http.StatusOK},
		{name: "student can ask basic RAG", method: http.MethodPost, path: "/knowledge-chat/s1", want: http.StatusOK},
		{name: "student can use knowledge search", method: http.MethodPost, path: "/knowledge-search", want: http.StatusOK},
		{name: "student can list authorized class resources", method: http.MethodGet, path: "/exam/resources", want: http.StatusOK},
		{name: "student cannot create KB", method: http.MethodPost, path: "/knowledge-bases", want: http.StatusForbidden},
		{name: "student cannot use agent chat", method: http.MethodPost, path: "/agent-chat/s1", want: http.StatusForbidden},
		{name: "student cannot bind KB resources", method: http.MethodPost, path: "/exam/resources/knowledge-bases/kb-1/bind", want: http.StatusForbidden},
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

func TestExamResourceRouteGuardSourceMatrix(t *testing.T) {
	sourceBytes, err := os.ReadFile("exam.go")
	if err != nil {
		t.Fatalf("read exam.go: %v", err)
	}
	source := string(sourceBytes)

	mustContainAll(t, source, []string{
		`exam.GET("/resources", g.Viewer(), resourceHandler.ListResources)`,
		`exam.POST("/resources/knowledge-bases/:kb_id/bind", g.Contributor(), resourceHandler.BindKnowledgeBase)`,
	})
}

func TestExamQuestionDraftRouteGuardSourceMatrix(t *testing.T) {
	sourceBytes, err := os.ReadFile("exam.go")
	if err != nil {
		t.Fatalf("read exam.go: %v", err)
	}
	source := string(sourceBytes)

	mustContainAll(t, source, []string{
		`exam.POST("/structuring-tasks/:task_id/extract", g.Contributor(), questionDraftHandler.ExtractDrafts)`,
		`exam.GET("/structuring-tasks/:task_id/drafts", g.Contributor(), questionDraftHandler.ListDrafts)`,
		`exam.PATCH("/question-drafts/:draft_id", g.Contributor(), questionDraftHandler.UpdateDraft)`,
		`exam.POST("/question-drafts/:draft_id/approve", g.Contributor(), questionDraftHandler.ApproveDraft)`,
		`exam.POST("/question-drafts/:draft_id/reject", g.Contributor(), questionDraftHandler.RejectDraft)`,
		`exam.GET("/question-banks/:bank_id/questions", g.Viewer(), questionHandler.ListQuestionDetails)`,
	})
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

func TestOrganizationSharedSpaceRoutesStayViewerAccessible(t *testing.T) {
	sourceBytes, err := os.ReadFile("router.go")
	if err != nil {
		t.Fatalf("read router.go: %v", err)
	}
	source := string(sourceBytes)

	mustContainAll(t, source, []string{
		`orgs.POST("", g.Viewer(), orgHandler.CreateOrganization)`,
		`orgs.POST("/join", g.Viewer(), orgHandler.JoinByInviteCode)`,
		`orgs.POST("/join-request", g.Viewer(), orgHandler.SubmitJoinRequest)`,
		`orgs.POST("/join-by-id", g.Viewer(), orgHandler.JoinByOrganizationID)`,
		`orgs.POST("/:id/invite-code", g.Viewer(), orgHandler.GenerateInviteCode)`,
		`orgs.GET("/:id/join-requests", g.Viewer(), orgHandler.ListJoinRequests)`,
		`orgs.PUT("/:id/join-requests/:request_id/review", g.Viewer(), orgHandler.ReviewJoinRequest)`,
	})
}

func TestBillingRouteGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	enabled := true
	guards := &rbacGuards{
		cfg: &config.Config{Tenant: &config.TenantConfig{EnableRBAC: &enabled}},
	}

	router := gin.New()
	router.GET("/billing/me", guards.Viewer(), okHandler)
	router.GET("/system/admin/billing/plans", guards.SystemAdmin(), okHandler)

	tests := []struct {
		name        string
		path        string
		systemAdmin bool
		want        int
	}{
		{name: "viewer can read own billing status", path: "/billing/me", want: http.StatusOK},
		{name: "normal user cannot read platform billing plans", path: "/system/admin/billing/plans", want: http.StatusForbidden},
		{name: "system admin can read platform billing plans", path: "/system/admin/billing/plans", systemAdmin: true, want: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			ctx := context.WithValue(req.Context(), types.TenantRoleContextKey, types.TenantRoleViewer)
			ctx = context.WithValue(ctx, types.UserIDContextKey, "user-1")
			ctx = context.WithValue(ctx, types.SystemAdminContextKey, tt.systemAdmin)
			req = req.WithContext(ctx)
			router.ServeHTTP(recorder, req)
			if recorder.Code != tt.want {
				t.Fatalf("GET %s status = %d, want %d", tt.path, recorder.Code, tt.want)
			}
		})
	}
}

func TestBillingRouteGuardSourceMatrix(t *testing.T) {
	sourceBytes, err := os.ReadFile("billing.go")
	if err != nil {
		t.Fatalf("read billing.go: %v", err)
	}
	source := string(sourceBytes)

	mustContainAll(t, source, []string{
		`r.GET("/billing/me", g.Viewer(), billingHandler.GetMine)`,
		`admin := r.Group("/system/admin/billing", g.SystemAdmin())`,
		`admin.POST("/plans", billingHandler.CreatePlan)`,
		`admin.PUT("/subscriptions/:tenant_id", billingHandler.UpsertTenantSubscription)`,
		`admin.POST("/orders", billingHandler.CreateOrder)`,
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
