package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExamDomainHandler struct {
	domainService interfaces.ExamDomainService
}

func NewExamDomainHandler(domainService interfaces.ExamDomainService) *ExamDomainHandler {
	return &ExamDomainHandler{domainService: domainService}
}

func (h *ExamDomainHandler) ListDomains(c *gin.Context) {
	ctx := c.Request.Context()
	domains, err := h.domainService.ListDomains(ctx)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam domains: %v", err)
		writeExamError(c, err, "Failed to list exam domains")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": domains})
}

func (h *ExamDomainHandler) ListSubjects(c *gin.Context) {
	ctx := c.Request.Context()
	domainID := c.Param("domain_id")
	subjects, err := h.domainService.ListSubjects(ctx, domainID)
	if err != nil {
		logger.Errorf(ctx, "Failed to list exam subjects: %v", err)
		writeExamError(c, err, "Failed to list exam subjects")
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": subjects})
}
