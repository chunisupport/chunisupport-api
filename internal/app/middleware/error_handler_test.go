package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/username/usernametest"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomHTTPErrorHandler_メンテナンス応答へ専用ヘッダーを付ける(t *testing.T) {
	// Given
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/songs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	CustomHTTPErrorHandler(c, apierror.ErrMaintenanceMode)

	// Then
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "60", rec.Header().Get(echo.HeaderRetryAfter))
	assert.Equal(t, "no-store", rec.Header().Get(echo.HeaderCacheControl))
	assert.JSONEq(t, `{
		"error": {
			"status": 503,
			"code": "maintenance_mode"
		}
	}`, rec.Body.String())

	var response apierror.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, apierror.CodeMaintenanceMode, response.Error.Code)
	assert.Equal(t, http.StatusServiceUnavailable, response.Error.Status)
	assert.Equal(t, 60, info.MaintenanceRetryAfterSeconds)
}

func TestLogError_メンテナンス応答はアプリケーションエラーログへ出さない(t *testing.T) {
	// Given
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/songs", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	logError(http.StatusServiceUnavailable, apierror.CodeMaintenanceMode, apierror.ErrMaintenanceMode, c)

	// Then
	assert.Empty(t, output.String())
}

func TestLogError_認証済みユーザーの識別子をログへ出す(t *testing.T) {
	// Given
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/me", nil)
	c := e.NewContext(req, httptest.NewRecorder())
	c.Set(contextKeyUserEntity, &entity.User{ID: 42, Username: usernametest.New(t, "testuser"), PlayerID: new(7)})

	// When
	logError(http.StatusInternalServerError, apierror.CodeInternalError, errors.New("boom"), c)

	// Then
	assert.Contains(t, output.String(), "level=ERROR")
	assert.Contains(t, output.String(), "user_id=42")
	assert.Contains(t, output.String(), "username=testuser")
	assert.Contains(t, output.String(), "player_id=7")
}

func TestCustomHTTPErrorHandler_フレームワークエラーをステータスに応じたコードへ変換する(t *testing.T) {
	tests := []struct {
		name string
		// Given: フレームワークが返すステータス
		status int
		// Then: 期待するレスポンス本文全体（仕様から求めた固定値）
		expectedBody string
	}{
		{name: "400はリクエスト形式不正", status: http.StatusBadRequest, expectedBody: `{"error":{"status":400,"code":"bad_request"}}`},
		{name: "401は認証が必要", status: http.StatusUnauthorized, expectedBody: `{"error":{"status":401,"code":"unauthorized"}}`},
		{name: "403は権限不足", status: http.StatusForbidden, expectedBody: `{"error":{"status":403,"code":"forbidden"}}`},
		{name: "404はエンドポイント未検出", status: http.StatusNotFound, expectedBody: `{"error":{"status":404,"code":"not_found"}}`},
		{name: "405はメソッド不正", status: http.StatusMethodNotAllowed, expectedBody: `{"error":{"status":405,"code":"method_not_allowed"}}`},
		{name: "409は競合", status: http.StatusConflict, expectedBody: `{"error":{"status":409,"code":"conflict"}}`},
		{name: "413はペイロード過大", status: http.StatusRequestEntityTooLarge, expectedBody: `{"error":{"status":413,"code":"payload_too_large"}}`},
		{name: "415はメディア種別不正", status: http.StatusUnsupportedMediaType, expectedBody: `{"error":{"status":415,"code":"unsupported_media_type"}}`},
		{name: "422はバリデーション失敗", status: http.StatusUnprocessableEntity, expectedBody: `{"error":{"status":422,"code":"validation_failed"}}`},
		{name: "429はレート制限", status: http.StatusTooManyRequests, expectedBody: `{"error":{"status":429,"code":"too_many_requests"}}`},
		{name: "503はサービス利用不可", status: http.StatusServiceUnavailable, expectedBody: `{"error":{"status":503,"code":"service_unavailable"}}`},
		{name: "対応表に無い4xxはリクエスト形式不正", status: http.StatusTeapot, expectedBody: `{"error":{"status":418,"code":"bad_request"}}`},
		{name: "対応表に無い5xxはサーバー内部エラー", status: http.StatusBadGateway, expectedBody: `{"error":{"status":502,"code":"internal_error"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			rec := httptest.NewRecorder()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
			// フレームワーク側のメッセージが利用者へ漏れないことも確かめるため、あえて内部向け文言を付ける。
			err := echo.NewHTTPError(tt.status, "内部向けの詳細")

			// When
			CustomHTTPErrorHandler(c, err)

			// Then
			assert.Equal(t, tt.status, rec.Code)
			assert.JSONEq(t, tt.expectedBody, rec.Body.String())
		})
	}
}

func TestCustomHTTPErrorHandler_素のエラーは内部エラーを返し原因を公開しない(t *testing.T) {
	// Given
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	const internalDetail = "機微な内部情報: password=secret"
	err := errors.New(internalDetail)

	// When
	CustomHTTPErrorHandler(c, err)

	// Then
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"error":{"status":500,"code":"internal_error"}}`, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), internalDetail)
}

func TestCustomHTTPErrorHandler_書き出し済みなら本文を上書きしない(t *testing.T) {
	// Given
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	require.NoError(t, c.String(http.StatusOK, "先に書いた応答"))

	// When
	CustomHTTPErrorHandler(c, errors.New("後から来たエラー"))

	// Then
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "先に書いた応答", rec.Body.String())
}

func TestCustomHTTPErrorHandler_クライアント切断は499としてINFOログに出す(t *testing.T) {
	// Given
	var output bytes.Buffer
	originalLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
	})

	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := apierror.ErrInternalError.WithInternal(errors.Join(usecase.ErrInternalError, context.Canceled))

	// When
	CustomHTTPErrorHandler(c, err)

	// Then
	assert.Equal(t, info.StatusClientClosedRequest, rec.Code)
	assert.Empty(t, rec.Body.String())
	assert.Contains(t, output.String(), "level=INFO")
	assert.Contains(t, output.String(), `msg="HTTP request canceled by client"`)
	assert.Contains(t, output.String(), "status=499")
	assert.NotContains(t, output.String(), "code=internal_error")
}
