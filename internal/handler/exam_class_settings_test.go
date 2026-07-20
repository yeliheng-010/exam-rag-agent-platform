package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExamClassSettingsListFilter(t *testing.T) {
	filters := make([]types.ListExamClassesFilter, 0, 2)
	stub := &stubExamClassSettingsService{
		listClasses: func(_ context.Context, _ uint64, _ string, filter types.ListExamClassesFilter) ([]*types.ExamClass, error) {
			filters = append(filters, filter)
			return []*types.ExamClass{}, nil
		},
	}
	router := newExamClassSettingsTestRouter(stub)

	require.Equal(t, http.StatusOK, performExamClassSettingsRequest(router, http.MethodGet, "/classes", "").Code)
	require.Equal(t, http.StatusOK, performExamClassSettingsRequest(router, http.MethodGet, "/classes?include_archived=true", "").Code)
	require.Equal(t, []types.ListExamClassesFilter{{}, {IncludeArchived: true}}, filters)
	require.Equal(t, http.StatusBadRequest, performExamClassSettingsRequest(router, http.MethodGet, "/classes?include_archived=invalid", "").Code)
}

func TestExamClassSettingsWriteEndpoints(t *testing.T) {
	var updateRequest *types.UpdateExamClassRequest
	calls := make([]string, 0, 3)
	stub := &stubExamClassSettingsService{
		updateClass: func(_ context.Context, tenantID uint64, userID, classID string, req *types.UpdateExamClassRequest) (*types.ExamClass, error) {
			require.Equal(t, uint64(7), tenantID)
			require.Equal(t, "owner", userID)
			require.Equal(t, "class-1", classID)
			updateRequest = req
			calls = append(calls, "update")
			return &types.ExamClass{ID: classID, Name: req.Name}, nil
		},
		archiveClass: func(_ context.Context, _ uint64, _ string, classID string) (*types.ExamClass, error) {
			calls = append(calls, "archive")
			return &types.ExamClass{ID: classID, Status: types.ExamClassStatusArchived}, nil
		},
		restoreClass: func(_ context.Context, _ uint64, _ string, classID string) (*types.ExamClass, error) {
			calls = append(calls, "restore")
			return &types.ExamClass{ID: classID, Status: types.ExamClassStatusActive}, nil
		},
	}
	router := newExamClassSettingsTestRouter(stub)

	response := performExamClassSettingsRequest(
		router, http.MethodPut, "/classes/class-1", `{"name":"高三一班","description":"冲刺班","member_limit":60}`,
	)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, &types.UpdateExamClassRequest{Name: "高三一班", Description: "冲刺班", MemberLimit: 60}, updateRequest)
	require.Equal(t, http.StatusOK, performExamClassSettingsRequest(router, http.MethodPost, "/classes/class-1/archive", "").Code)
	require.Equal(t, http.StatusOK, performExamClassSettingsRequest(router, http.MethodPost, "/classes/class-1/restore", "").Code)
	require.Equal(t, []string{"update", "archive", "restore"}, calls)
}

func TestExamClassSettingsErrorStatusCodes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "invalid", err: service.ErrExamInvalidRequest, want: http.StatusBadRequest},
		{name: "forbidden", err: service.ErrExamPermissionDenied, want: http.StatusForbidden},
		{name: "not found", err: service.ErrExamNotFound, want: http.StatusNotFound},
		{name: "conflict", err: service.ErrExamStateConflict, want: http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubExamClassSettingsService{
				updateClass: func(context.Context, uint64, string, string, *types.UpdateExamClassRequest) (*types.ExamClass, error) {
					return nil, tt.err
				},
			}
			response := performExamClassSettingsRequest(
				newExamClassSettingsTestRouter(stub), http.MethodPut, "/classes/class-1", `{"name":"班级","member_limit":50}`,
			)
			require.Equal(t, tt.want, response.Code, response.Body.String())
		})
	}

	response := performExamClassSettingsRequest(
		newExamClassSettingsTestRouter(&stubExamClassSettingsService{}), http.MethodPut, "/classes/class-1", `{`,
	)
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func newExamClassSettingsTestRouter(stub *stubExamClassSettingsService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	router.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(7))
		c.Set(types.UserIDContextKey.String(), "owner")
		c.Next()
	})
	handler := NewExamClassHandler(stub)
	router.GET("/classes", handler.ListClasses)
	router.PUT("/classes/:class_id", handler.UpdateClass)
	router.POST("/classes/:class_id/archive", handler.ArchiveClass)
	router.POST("/classes/:class_id/restore", handler.RestoreClass)
	return router
}

func performExamClassSettingsRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

type stubExamClassSettingsService struct {
	listClasses  func(context.Context, uint64, string, types.ListExamClassesFilter) ([]*types.ExamClass, error)
	updateClass  func(context.Context, uint64, string, string, *types.UpdateExamClassRequest) (*types.ExamClass, error)
	archiveClass func(context.Context, uint64, string, string) (*types.ExamClass, error)
	restoreClass func(context.Context, uint64, string, string) (*types.ExamClass, error)
}

func (s *stubExamClassSettingsService) ListClasses(ctx context.Context, tenantID uint64, userID string, filter types.ListExamClassesFilter) ([]*types.ExamClass, error) {
	if s.listClasses == nil {
		return []*types.ExamClass{}, nil
	}
	return s.listClasses(ctx, tenantID, userID, filter)
}

func (s *stubExamClassSettingsService) UpdateClass(ctx context.Context, tenantID uint64, userID, classID string, req *types.UpdateExamClassRequest) (*types.ExamClass, error) {
	if s.updateClass == nil {
		return &types.ExamClass{ID: classID}, nil
	}
	return s.updateClass(ctx, tenantID, userID, classID, req)
}

func (s *stubExamClassSettingsService) ArchiveClass(ctx context.Context, tenantID uint64, userID, classID string) (*types.ExamClass, error) {
	if s.archiveClass == nil {
		return &types.ExamClass{ID: classID}, nil
	}
	return s.archiveClass(ctx, tenantID, userID, classID)
}

func (s *stubExamClassSettingsService) RestoreClass(ctx context.Context, tenantID uint64, userID, classID string) (*types.ExamClass, error) {
	if s.restoreClass == nil {
		return &types.ExamClass{ID: classID}, nil
	}
	return s.restoreClass(ctx, tenantID, userID, classID)
}

func (*stubExamClassSettingsService) CreateClass(context.Context, uint64, string, *types.CreateExamClassRequest) (*types.ExamClass, error) {
	return nil, nil
}
func (*stubExamClassSettingsService) GetClass(context.Context, uint64, string, string) (*types.ExamClass, error) {
	return nil, nil
}
func (*stubExamClassSettingsService) RequestJoinClass(context.Context, uint64, string, *types.JoinExamClassRequest) (*types.ExamClassMember, error) {
	return nil, nil
}
func (*stubExamClassSettingsService) ListClassMembers(context.Context, uint64, string, string) ([]*types.ExamClassMember, error) {
	return nil, nil
}
func (*stubExamClassSettingsService) ApproveClassMember(context.Context, uint64, string, string, string) (*types.ExamClassMember, error) {
	return nil, nil
}
func (*stubExamClassSettingsService) RejectClassMember(context.Context, uint64, string, string, string) (*types.ExamClassMember, error) {
	return nil, nil
}
func (*stubExamClassSettingsService) CanAccessClass(context.Context, uint64, string, string) (bool, error) {
	return false, nil
}
func (*stubExamClassSettingsService) CanWriteClass(context.Context, uint64, string, string) (bool, error) {
	return false, nil
}
