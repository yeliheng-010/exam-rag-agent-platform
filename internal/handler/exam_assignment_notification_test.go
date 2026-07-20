package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExamAssignmentNotificationHandlerRejectsInvalidReminderJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPost,
		"/classes/class-1/assignments/assignment-1/reminders",
		strings.NewReader(`{"recipient_user_ids":[`),
	)
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler := NewExamAssignmentHandler(&stubAssignmentNotificationService{})

	handler.SendAssignmentReminders(ctx)

	require.Len(t, ctx.Errors, 1)
	appErr, ok := ctx.Errors[0].Err.(*apperrors.AppError)
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, appErr.HTTPCode)
}

func TestExamAssignmentNotificationHandlerSendsReminders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &stubAssignmentNotificationService{
		result: &types.SendExamAssignmentReminderResult{SentCount: 1, SentUserIDs: []string{"student-1"}},
	}
	handler := NewExamAssignmentHandler(service)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(10000))
		c.Set(types.UserIDContextKey.String(), "teacher-1")
		c.Next()
	})
	router.POST("/classes/:class_id/assignments/:assignment_id/reminders", handler.SendAssignmentReminders)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/classes/class-1/assignments/assignment-1/reminders",
		strings.NewReader(`{"recipient_user_ids":["student-1"]}`),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, uint64(10000), service.tenantID)
	require.Equal(t, "teacher-1", service.userID)
	require.Equal(t, "class-1", service.classID)
	require.Equal(t, "assignment-1", service.assignmentID)
	require.Equal(t, []string{"student-1"}, service.request.RecipientUserIDs)
	require.Contains(t, recorder.Body.String(), `"sent_count":1`)
}

type stubAssignmentNotificationService struct {
	interfaces.ExamAssignmentService
	tenantID     uint64
	userID       string
	classID      string
	assignmentID string
	request      *types.SendExamAssignmentReminderRequest
	result       *types.SendExamAssignmentReminderResult
}

func (s *stubAssignmentNotificationService) SendAssignmentReminders(
	_ context.Context,
	tenantID uint64,
	userID string,
	classID string,
	assignmentID string,
	request *types.SendExamAssignmentReminderRequest,
) (*types.SendExamAssignmentReminderResult, error) {
	s.tenantID = tenantID
	s.userID = userID
	s.classID = classID
	s.assignmentID = assignmentID
	s.request = request
	return s.result, nil
}
