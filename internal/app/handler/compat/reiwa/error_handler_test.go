package reiwa

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReiwaErrorHandlerMiddleware_メンテナンス中も互換503形式を維持する(t *testing.T) {
	// Given
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/compat/reiwa/1/chunithm_record/original", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handler := ReiwaErrorHandlerMiddleware()(func(c *echo.Context) error {
		c.Response().Header().Set(echo.HeaderRetryAfter, "60")
		c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
		return apierror.ErrMaintenanceMode
	})

	// When
	err := handler(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "60", rec.Header().Get(echo.HeaderRetryAfter))
	assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))

	var response ReiwaErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, http.StatusServiceUnavailable, response.Error.Code)
	assert.Equal(t, "service unavailable.", response.Error.Message)
	assert.Empty(t, response.Error.AdditionalMessage)
}

func TestLogReiwaError_メンテナンス応答はアプリケーションエラーログへ出さない(t *testing.T) {
	// Given
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/compat/reiwa/1/chunithm_record/original", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	logReiwaError(http.StatusServiceUnavailable, apierror.ErrMaintenanceMode, c)

	// Then
	assert.Empty(t, output.String())
}
