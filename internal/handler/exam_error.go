package handler

import (
	"errors"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/gin-gonic/gin"
)

func writeExamError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrExamInvalidRequest):
		c.Error(apperrors.NewValidationError("Invalid request parameters"))
	case errors.Is(err, service.ErrExamPermissionDenied):
		c.Error(apperrors.NewForbiddenError("Permission denied"))
	case errors.Is(err, service.ErrExamNotFound),
		errors.Is(err, repository.ErrExamDomainNotFound),
		errors.Is(err, repository.ErrExamSubjectNotFound),
		errors.Is(err, repository.ErrExamSpaceNotFound),
		errors.Is(err, repository.ErrExamClassNotFound),
		errors.Is(err, repository.ErrQuestionBankNotFound),
		errors.Is(err, repository.ErrExamResourceNotFound):
		c.Error(apperrors.NewNotFoundError("Exam resource not found"))
	default:
		c.Error(apperrors.NewInternalServerError(fallback).WithDetails(err.Error()))
	}
}
