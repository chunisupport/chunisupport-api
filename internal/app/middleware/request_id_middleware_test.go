package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/infra/logger"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware_外部IDの受理と置換(t *testing.T) {
	valid128 := "a" + strings.Repeat("b", info.RequestIDMaxLength-1)
	tooLong := strings.Repeat("a", info.RequestIDMaxLength+1)

	tests := []struct {
		name         string
		setHeader    bool
		headerName   string
		incoming     string
		extraValue   string
		wantID       string
		wantGenerate bool
	}{
		{
			name:         "ヘッダーがない場合はUUIDを生成する",
			wantGenerate: true,
		},
		{
			name:       "妥当なIDは受理する",
			setHeader:  true,
			headerName: "X-Request-ID",
			incoming:   "abc-123_DEF.9",
			wantID:     "abc-123_DEF.9",
		},
		{
			name:       "前後の空白は除いて受理する",
			setHeader:  true,
			headerName: echo.HeaderXRequestID,
			incoming:   "  trace-1  ",
			wantID:     "trace-1",
		},
		{
			name:       "128文字のIDは受理する",
			setHeader:  true,
			headerName: echo.HeaderXRequestID,
			incoming:   valid128,
			wantID:     valid128,
		},
		{
			name:         "空文字はUUIDへ置き換える",
			setHeader:    true,
			headerName:   echo.HeaderXRequestID,
			incoming:     "",
			wantGenerate: true,
		},
		{
			name:         "長すぎるIDはUUIDへ置き換える",
			setHeader:    true,
			headerName:   echo.HeaderXRequestID,
			incoming:     tooLong,
			wantGenerate: true,
		},
		{
			name:         "空白を含むIDはUUIDへ置き換える",
			setHeader:    true,
			headerName:   echo.HeaderXRequestID,
			incoming:     "bad id",
			wantGenerate: true,
		},
		{
			name:         "改行を含むIDはUUIDへ置き換える",
			setHeader:    true,
			headerName:   echo.HeaderXRequestID,
			incoming:     "bad\nid",
			wantGenerate: true,
		},
		{
			name:         "記号始まりはUUIDへ置き換える",
			setHeader:    true,
			headerName:   echo.HeaderXRequestID,
			incoming:     "-abc",
			wantGenerate: true,
		},
		{
			name:       "複数値のときは先頭の妥当なIDだけを残す",
			setHeader:  true,
			headerName: echo.HeaderXRequestID,
			incoming:   "valid-id",
			extraValue: "second\ninjection",
			wantID:     "valid-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.Use(RequestIDMiddleware())
			e.GET("/id", func(c *echo.Context) error {
				return c.String(http.StatusOK, logger.RequestIDFromContext(c.Request().Context()))
			})
			req := httptest.NewRequest(http.MethodGet, "/id", nil)
			if tt.setHeader {
				req.Header.Set(tt.headerName, tt.incoming)
				if tt.extraValue != "" {
					req.Header.Add(echo.HeaderXRequestID, tt.extraValue)
				}
			}
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			require.Equal(t, http.StatusOK, rec.Code)
			got := rec.Header().Get(echo.HeaderXRequestID)
			assert.Equal(t, got, rec.Body.String())
			assert.NotContains(t, got, "\n")
			if tt.wantGenerate {
				_, err := uuid.Parse(got)
				assert.NoError(t, err)
				assert.NotEqual(t, strings.TrimSpace(tt.incoming), got)
			} else {
				assert.Equal(t, tt.wantID, got)
			}
			if tt.extraValue != "" {
				assert.Equal(t, []string{tt.wantID}, rec.Header().Values(echo.HeaderXRequestID))
			}
		})
	}
}

func TestRequestIDMiddleware_レスポンスとログに同じIDを出す(t *testing.T) {
	// Given
	appLogPath, restoreLogger := useAppLogger(t)
	defer restoreLogger()

	var accessLog bytes.Buffer
	e := echo.New()
	e.Logger = slog.New(slog.NewTextHandler(&accessLog, nil))
	e.HTTPErrorHandler = CustomHTTPErrorHandler
	e.Use(RequestIDMiddleware())
	e.Use(echoMiddleware.RequestLogger())
	e.GET("/boom", func(c *echo.Context) error {
		slog.ErrorContext(c.Request().Context(), "handler failed")
		return apierror.ErrInternalError
	})
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.Header.Set("X-Request-ID", "client-req-1")
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "client-req-1", rec.Header().Get(echo.HeaderXRequestID))

	appLog := readLogFile(t, appLogPath)
	assert.Contains(t, appLog, "request_id=client-req-1")
	assert.Contains(t, appLog, "handler failed")
	assert.Contains(t, appLog, "HTTP error occurred")

	assert.Contains(t, accessLog.String(), "request_id=client-req-1")
	assert.Contains(t, accessLog.String(), "status=500")
}

func TestRequestIDMiddleware_不正なIDはログへ残さず生成IDで揃える(t *testing.T) {
	// Given
	appLogPath, restoreLogger := useAppLogger(t)
	defer restoreLogger()

	var accessLog bytes.Buffer
	e := echo.New()
	e.Logger = slog.New(slog.NewTextHandler(&accessLog, nil))
	e.HTTPErrorHandler = CustomHTTPErrorHandler
	e.Use(RequestIDMiddleware())
	e.Use(echoMiddleware.RequestLogger())
	e.GET("/boom", func(c *echo.Context) error {
		slog.ErrorContext(c.Request().Context(), "handler failed")
		return apierror.ErrInternalError
	})
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.Header.Set(echo.HeaderXRequestID, "bad id\ninjected")
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	got := rec.Header().Get(echo.HeaderXRequestID)
	_, err := uuid.Parse(got)
	require.NoError(t, err)

	appLog := readLogFile(t, appLogPath)
	accessText := accessLog.String()
	assert.Contains(t, appLog, "request_id="+got)
	assert.Contains(t, accessText, "request_id="+got)
	assert.NotContains(t, appLog, "bad id")
	assert.NotContains(t, accessText, "bad id")
	assert.NotContains(t, appLog, "injected")
	assert.NotContains(t, accessText, "injected")
}

func useAppLogger(t *testing.T) (string, func()) {
	t.Helper()

	appLogPath := filepath.Join(t.TempDir(), "app.log")
	handler, err := logger.NewHandler(config.Logging{
		Level:   "debug",
		AppFile: appLogPath,
		Stdout:  false,
	})
	require.NoError(t, err)

	original := slog.Default()
	slog.SetDefault(slog.New(handler))
	return appLogPath, func() {
		slog.SetDefault(original)
		assert.NoError(t, handler.Close())
	}
}

func readLogFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
