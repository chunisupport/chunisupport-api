package chunirec

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunirecErrorHandlerMiddleware_メンテナンス中も互換503形式を維持する(t *testing.T) {
	// Given
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/compat/chunirec/2.0/music/showall", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handler := ChunirecErrorHandlerMiddleware()(func(c *echo.Context) error {
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

	var response ChunirecErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, http.StatusServiceUnavailable, response.Error.Code)
	assert.Equal(t, "service unavailable.", response.Error.Message)
	assert.Empty(t, response.Error.AdditionalMessage)
}

func TestLogChunirecError_メンテナンス応答はアプリケーションエラーログへ出さない(t *testing.T) {
	// Given
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/compat/chunirec/2.0/music/showall", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	logChunirecError(http.StatusServiceUnavailable, apierror.ErrMaintenanceMode, c)

	// Then
	assert.Empty(t, output.String())
}

func TestLogChunirecError_認証済みユーザーの識別子をログへ出す(t *testing.T) {
	// Given
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/compat/chunirec/2.0/records/showall", nil)
	c := e.NewContext(req, httptest.NewRecorder())
	c.Set("userEntity", &entity.User{ID: 42, Username: usernametest.New(t, "testuser"), PlayerID: new(7)})

	// When
	logChunirecError(http.StatusInternalServerError, errors.New("boom"), c)

	// Then
	assert.Contains(t, output.String(), "level=ERROR")
	assert.Contains(t, output.String(), "user_id=42")
	assert.Contains(t, output.String(), "username=testuser")
	assert.Contains(t, output.String(), "player_id=7")
}

func TestChunirecErrorHandlerMiddleware_クライアント切断は499としてINFOログに出す(t *testing.T) {
	// Given
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/compat/chunirec/2.0/records/showall", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handler := ChunirecErrorHandlerMiddleware()(func(c *echo.Context) error {
		return apierror.ErrInternalError.WithInternal(context.Canceled)
	})

	// When
	err := handler(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, info.StatusClientClosedRequest, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Contains(t, output.String(), "level=INFO")
	assert.Contains(t, output.String(), `msg="Chunirec HTTP request canceled by client"`)
	assert.Contains(t, output.String(), "status=499")
}

func TestChunirecErrorHandlerMiddleware_Echoのエラーはステータスを維持する(t *testing.T) {
	tests := []struct {
		name string
		// Given: 後続処理が返すエラー
		err error
		// Then: 期待するステータスとメッセージ
		expectedStatus  int
		expectedMessage string
	}{
		{
			name:            "ルート不一致の404は404のまま返す",
			err:             echo.ErrNotFound,
			expectedStatus:  http.StatusNotFound,
			expectedMessage: "not found.",
		},
		{
			name:            "メソッド不一致の405は405のまま返す",
			err:             echo.ErrMethodNotAllowed,
			expectedStatus:  http.StatusMethodNotAllowed,
			expectedMessage: "the requested method is not allowed.",
		},
		{
			name:            "ラップされたHTTPErrorもステータスを維持する",
			err:             fmt.Errorf("wrap: %w", echo.NewHTTPError(http.StatusTooManyRequests, "")),
			expectedStatus:  http.StatusTooManyRequests,
			expectedMessage: "too many requests.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/compat/chunirec/2.0/music/show", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			handler := ChunirecErrorHandlerMiddleware()(func(c *echo.Context) error {
				return tt.err
			})

			// When
			err := handler(c)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)

			var response ChunirecErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
			assert.Equal(t, tt.expectedStatus, response.Error.Code)
			assert.Equal(t, tt.expectedMessage, response.Error.Message)
		})
	}
}
