package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeadersMiddleware_すべての応答にセキュリティヘッダーを付与する(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "正常応答", method: http.MethodGet, path: "/ok", wantStatus: http.StatusOK},
		{name: "ハンドラーのエラー応答", method: http.MethodGet, path: "/error", wantStatus: http.StatusInternalServerError},
		{name: "未登録パスの404", method: http.MethodGet, path: "/missing", wantStatus: http.StatusNotFound},
		{name: "メソッド不一致の405", method: http.MethodPost, path: "/ok", wantStatus: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			e.Use(newSecurityHeadersMiddleware())
			e.GET("/ok", func(c *echo.Context) error {
				return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
			})
			e.GET("/error", func(c *echo.Context) error {
				return errors.New("unexpected")
			})
			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "nosniff", rec.Header().Get(echo.HeaderXContentTypeOptions))
			assert.Equal(t, "DENY", rec.Header().Get(echo.HeaderXFrameOptions))
			assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", rec.Header().Get(echo.HeaderContentSecurityPolicy))
			assert.Equal(t, "no-referrer", rec.Header().Get(echo.HeaderReferrerPolicy))
			assert.Empty(t, rec.Header().Get(echo.HeaderXXSSProtection))
		})
	}
}

func TestNewRouter_メンテナンス応答とCORSプリフライトにもセキュリティヘッダーを付与する(t *testing.T) {
	// Given: メンテナンス中のルーター（応答を途中で返すミドルウェアを経由させる）
	database, err := sqlx.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})
	_, err = database.Exec(`
		CREATE TABLE system_maintenance (
			id INTEGER NOT NULL PRIMARY KEY,
			enabled BOOLEAN NOT NULL,
			comment TEXT NOT NULL,
			updated_by_user_id INTEGER NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	require.NoError(t, err)
	_, err = database.Exec(
		`INSERT INTO system_maintenance (id, enabled, comment, updated_by_user_id, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		1, true, "データ更新中です", nil, time.Date(2026, time.July, 26, 3, 30, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	router, err := NewRouter(
		context.Background(),
		database,
		config.Config{DataTransferHMACSecret: testDataTransferHMACSecret,
			Location: time.FixedZone("JST", 9*60*60),
			CORS:     config.CORS{AllowOrigins: []string{"https://example.com"}},
		},
		nil, nil, tokenVerifierWithRecentSignIn{}, nil, nil, nil,
	)
	require.NoError(t, err)

	tests := []struct {
		name       string
		method     string
		path       string
		preflight  bool
		wantStatus int
	}{
		{name: "ヘルスチェック", method: http.MethodGet, path: "/healthz", wantStatus: http.StatusNoContent},
		{name: "メンテナンスの503", method: http.MethodGet, path: "/version", wantStatus: http.StatusServiceUnavailable},
		{name: "CORSプリフライト", method: http.MethodOptions, path: "/healthz", preflight: true, wantStatus: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			if tt.preflight {
				req.Header.Set(echo.HeaderOrigin, "https://example.com")
				req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
			}
			rec := httptest.NewRecorder()

			// When
			router.ServeHTTP(rec, req)

			// Then
			require.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, "nosniff", rec.Header().Get(echo.HeaderXContentTypeOptions))
			assert.Equal(t, "DENY", rec.Header().Get(echo.HeaderXFrameOptions))
			assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", rec.Header().Get(echo.HeaderContentSecurityPolicy))
			assert.Equal(t, "no-referrer", rec.Header().Get(echo.HeaderReferrerPolicy))
		})
	}
}

func TestSecurityHeadersMiddleware_HSTSはCloudflareに任せて付与しない(t *testing.T) {
	// Given
	e := echo.New()
	e.Use(newSecurityHeadersMiddleware())
	e.GET("/ok", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ok", nil)
	req.Header.Set(echo.HeaderXForwardedProto, "https")
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	assert.Empty(t, rec.Header().Get(echo.HeaderStrictTransportSecurity))
}
