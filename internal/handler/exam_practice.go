package handler

import (
	"net/http"
	"strconv"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamPracticeHandler struct {
	practiceService interfaces.ExamPracticeService
}

func NewExamPracticeHandler(practiceService interfaces.ExamPracticeService) *ExamPracticeHandler {
	return &ExamPracticeHandler{practiceService: practiceService}
}

func (h *ExamPracticeHandler) ListQuestionGroups(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	limit, _ := strconv.Atoi(c.Query("limit"))
	filter := types.ListPracticeQuestionGroupsFilter{
		SpaceID:   c.Query("space_id"),
		DomainID:  c.Query("domain_id"),
		SubjectID: c.Query("subject_id"),
		Limit:     limit,
	}

	groups, err := h.practiceService.ListQuestionGroups(ctx, tenantID, userID, filter)
	if err != nil {
		logger.Errorf(ctx, "Failed to list practice question groups: %v", err)
		writeExamError(c, err, "Failed to list practice question groups")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": groups})
}

func (h *ExamPracticeHandler) ListAttempts(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	limit, _ := strconv.Atoi(c.Query("limit"))
	filter := types.ListPracticeAttemptsFilter{
		SpaceID: c.Query("space_id"),
		GroupID: c.Query("group_id"),
		Limit:   limit,
	}

	attempts, err := h.practiceService.ListAttempts(ctx, tenantID, userID, filter)
	if err != nil {
		logger.Errorf(ctx, "Failed to list practice attempts: %v", err)
		writeExamError(c, err, "Failed to list practice attempts")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": attempts})
}

func (h *ExamPracticeHandler) GetAttempt(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	attempt, err := h.practiceService.GetAttemptDetail(ctx, tenantID, userID, c.Param("attempt_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to get practice attempt: %v", err)
		writeExamError(c, err, "Failed to get practice attempt")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": attempt})
}

func (h *ExamPracticeHandler) ListWrongQuestions(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	limit, _ := strconv.Atoi(c.Query("limit"))
	filter := types.ListWrongQuestionsFilter{
		SpaceID: c.Query("space_id"),
		GroupID: c.Query("group_id"),
		Limit:   limit,
	}

	items, err := h.practiceService.ListWrongQuestions(ctx, tenantID, userID, filter)
	if err != nil {
		logger.Errorf(ctx, "Failed to list wrong questions: %v", err)
		writeExamError(c, err, "Failed to list wrong questions")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": items})
}

func (h *ExamPracticeHandler) GetQuestionGroup(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	group, err := h.practiceService.GetQuestionGroupDetail(ctx, tenantID, userID, c.Param("group_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to get practice question group: %v", err)
		writeExamError(c, err, "Failed to get practice question group")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": group})
}

func (h *ExamPracticeHandler) CreateAttempt(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	result, err := h.practiceService.CreateAttempt(ctx, tenantID, userID, c.Param("group_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to create practice attempt: %v", err)
		writeExamError(c, err, "Failed to create practice attempt")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": result})
}

func (h *ExamPracticeHandler) SubmitAnswer(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	var req types.SubmitPracticeAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewValidationError("Invalid request parameters").WithDetails(err.Error()))
		return
	}
	result, err := h.practiceService.SubmitAnswer(ctx, tenantID, userID, c.Param("attempt_id"), &req)
	if err != nil {
		logger.Errorf(ctx, "Failed to submit practice answer: %v", err)
		writeExamError(c, err, "Failed to submit practice answer")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (h *ExamPracticeHandler) CompleteAttempt(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetString(types.UserIDContextKey.String())
	tenantID := c.GetUint64(types.TenantIDContextKey.String())

	attempt, err := h.practiceService.CompleteAttempt(ctx, tenantID, userID, c.Param("attempt_id"))
	if err != nil {
		logger.Errorf(ctx, "Failed to complete practice attempt: %v", err)
		writeExamError(c, err, "Failed to complete practice attempt")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": attempt})
}
