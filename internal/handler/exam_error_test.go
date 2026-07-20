package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestWriteExamErrorMapsStateConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	writeExamError(ctx, service.ErrExamStateConflict, "fallback")

	require.Len(t, ctx.Errors, 1)
	appErr, ok := ctx.Errors[0].Err.(*apperrors.AppError)
	require.True(t, ok)
	require.Equal(t, apperrors.ErrConflict, appErr.Code)
	require.Equal(t, http.StatusConflict, appErr.HTTPCode)
}
