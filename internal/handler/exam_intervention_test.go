package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type stubExamInterventionHandlerService struct {
	interfaces.ExamInterventionService
	classErr error
}

func (s *stubExamInterventionHandlerService) RecommendClassPractice(
	context.Context,
	uint64,
	string,
	string,
	types.ExamPracticeRecommendationRequest,
) (*types.ExamPracticeRecommendationResult, error) {
	return nil, s.classErr
}

func TestExamInterventionHandlerMapsClassPermissionDeniedToForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	router.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(10000))
		c.Set(types.UserIDContextKey.String(), "student-1")
		c.Next()
	})
	h := NewExamInterventionHandler(&stubExamInterventionHandlerService{classErr: service.ErrExamPermissionDenied})
	router.GET("/api/v1/exam/classes/:class_id/practice-recommendations", h.GetClassRecommendations)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/exam/classes/class-1/practice-recommendations", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusForbidden, w.Body.String())
	}
}
