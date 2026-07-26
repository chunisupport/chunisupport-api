package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalCORS_対象エンドポイントのみ追加オリジンを許可する(t *testing.T) {
	cfg := config.Config{
		CORS: config.CORS{
			AllowOrigins:     []string{"https://chunisupport.example.com"},
			AllowCredentials: true,
			MaxAge:           600,
		},
	}

	tests := []struct {
		name          string
		path          string
		requestMethod string
		setupRoute    func(e *echo.Echo)
		wantAllow     string
		wantAllowCred string
	}{
		{
			name:          "外部向けヘルスチェックでは追加オリジンを許可する",
			path:          "/healthz",
			requestMethod: http.MethodGet,
			setupRoute: func(e *echo.Echo) {
				healthzCORS := echoMiddleware.CORSWithConfig(newExternalCORSConfig(cfg))
				e.OPTIONS("/healthz", func(c *echo.Context) error {
					return c.NoContent(http.StatusNoContent)
				}, healthzCORS)
			},
			wantAllow:     "https://new.chunithm-net.com",
			wantAllowCred: "true",
		},
		{
			name:          "ルートでは追加オリジンを許可しない",
			path:          "/",
			requestMethod: http.MethodGet,
			setupRoute: func(e *echo.Echo) {
				e.OPTIONS("/", func(c *echo.Context) error {
					return c.NoContent(http.StatusNoContent)
				})
			},
			wantAllow:     "",
			wantAllowCred: "",
		},
		{
			name:          "一時保存エンドポイントでは追加オリジンを許可する",
			path:          "/internal/player-data/temp",
			requestMethod: http.MethodPost,
			setupRoute: func(e *echo.Echo) {
				tempDataCORS := echoMiddleware.CORSWithConfig(newExternalCORSConfig(cfg))
				e.OPTIONS("/internal/player-data/temp", func(c *echo.Context) error {
					return c.NoContent(http.StatusNoContent)
				}, tempDataCORS)
			},
			wantAllow:     "https://new.chunithm-net.com",
			wantAllowCred: "true",
		},
		{
			name:          "他のエンドポイントでは追加オリジンを許可しない",
			path:          "/internal/users/sample",
			requestMethod: http.MethodPost,
			setupRoute: func(e *echo.Echo) {
				e.OPTIONS("/internal/users/:username", func(c *echo.Context) error {
					return c.NoContent(http.StatusNoContent)
				})
			},
			wantAllow:     "",
			wantAllowCred: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Use(echoMiddleware.CORSWithConfig(newDefaultCORSConfig(cfg)))
			tt.setupRoute(e)

			req := httptest.NewRequest(http.MethodOptions, tt.path, nil)
			req.Header.Set(echo.HeaderOrigin, "https://new.chunithm-net.com")
			req.Header.Set(echo.HeaderAccessControlRequestMethod, tt.requestMethod)
			req.Header.Set(echo.HeaderAccessControlRequestHeaders, strings.Join([]string{
				echo.HeaderContentType,
				echo.HeaderContentEncoding,
				"X-Reauth-Token",
			}, ", "))
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantAllow, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
			assert.Equal(t, tt.wantAllowCred, rec.Header().Get(echo.HeaderAccessControlAllowCredentials))
			if tt.wantAllow != "" {
				assert.Contains(t, rec.Header().Get(echo.HeaderAccessControlAllowHeaders), echo.HeaderContentEncoding)
				assert.Contains(t, rec.Header().Get(echo.HeaderAccessControlAllowHeaders), "X-Reauth-Token")
			}
		})
	}
}

func TestTemporaryPlayerDataCORS_内部グループでも対象パスだけ追加オリジンを許可する(t *testing.T) {
	// Given
	cfg := config.Config{
		CORS: config.CORS{
			AllowOrigins: []string{"https://chunisupport.example.com"},
		},
	}
	e := echo.New()
	e.Use(echoMiddleware.CORSWithConfig(newDefaultCORSConfig(cfg)))
	internal := e.Group("/internal")
	internal.Use(echoMiddleware.CORSWithConfig(newTemporaryPlayerDataCORSConfig(cfg)))
	internal.OPTIONS("/player-data/temp", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	internal.OPTIONS("/me", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	tests := []struct {
		name      string
		path      string
		wantAllow string
	}{
		{
			name:      "一時保存APIは追加オリジンを許可する",
			path:      "/internal/player-data/temp",
			wantAllow: info.ExternalCORSAllowOrigin,
		},
		{
			name: "他の内部APIへ追加オリジンを広げない",
			path: "/internal/me",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, tt.path, nil)
			req.Header.Set(echo.HeaderOrigin, info.ExternalCORSAllowOrigin)
			req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPost)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			assert.Equal(t, tt.wantAllow, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
		})
	}
}

func TestCORSAllowOrigins_ワイルドカード入りオリジンを許可する(t *testing.T) {
	cfg := config.Config{
		CORS: config.CORS{
			AllowOrigins: []string{
				"https://chunisupport.example.com",
				"https://*.chunisupport.pages.dev",
			},
			AllowCredentials: true,
			MaxAge:           600,
		},
	}

	tests := []struct {
		name      string
		origin    string
		wantAllow string
	}{
		{
			name:      "完全一致のオリジンを許可する",
			origin:    "https://chunisupport.example.com",
			wantAllow: "https://chunisupport.example.com",
		},
		{
			name:      "Pagesのプレビューサブドメインを許可する",
			origin:    "https://feature-branch.chunisupport.pages.dev",
			wantAllow: "https://feature-branch.chunisupport.pages.dev",
		},
		{
			name:   "Pagesのルートドメインは許可しない",
			origin: "https://chunisupport.pages.dev",
		},
		{
			name:   "別スキームは許可しない",
			origin: "http://feature-branch.chunisupport.pages.dev",
		},
		{
			name:   "別ドメインは許可しない",
			origin: "https://feature-branch.example.pages.dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Use(echoMiddleware.CORSWithConfig(newDefaultCORSConfig(cfg)))
			e.OPTIONS("/internal/me", func(c *echo.Context) error {
				return c.NoContent(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodOptions, "/internal/me", nil)
			req.Header.Set(echo.HeaderOrigin, tt.origin)
			req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodGet)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantAllow, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
			if tt.wantAllow == "" {
				assert.Empty(t, rec.Header().Get(echo.HeaderAccessControlAllowCredentials))
				return
			}
			assert.Equal(t, "true", rec.Header().Get(echo.HeaderAccessControlAllowCredentials))
		})
	}
}

func TestCORSAllowMethods_PATCHを許可する(t *testing.T) {
	// Given
	cfg := config.Config{CORS: config.CORS{AllowOrigins: []string{"https://chunisupport.example.com"}}}
	e := echo.New()
	e.Use(echoMiddleware.CORSWithConfig(newDefaultCORSConfig(cfg)))
	e.PATCH("/internal/auth/api-tokens/:id", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodOptions, "/internal/auth/api-tokens/1", nil)
	req.Header.Set(echo.HeaderOrigin, "https://chunisupport.example.com")
	req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPatch)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	assert.Equal(t, "https://chunisupport.example.com", rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	assert.Contains(t, rec.Header().Get(echo.HeaderAccessControlAllowMethods), http.MethodPatch)
}

func TestCORSExposeHeaders_RetryAfterを公開する(t *testing.T) {
	// Given
	cfg := config.Config{CORS: config.CORS{AllowOrigins: []string{"https://chunisupport.example.com"}}}

	// When
	corsConfig := newDefaultCORSConfig(cfg)

	// Then
	assert.Contains(t, corsConfig.ExposeHeaders, echo.HeaderRetryAfter)
}

func TestMaintenanceResponse_許可済みオリジンへCORSヘッダーを返す(t *testing.T) {
	// Given
	const origin = "https://chunisupport.example.com"
	cfg := config.Config{CORS: config.CORS{AllowOrigins: []string{origin}}}
	maintenance := stubMaintenanceUsecase{state: usecase.MaintenanceState{Enabled: true}}
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	e.Use(echoMiddleware.CORSWithConfig(newDefaultCORSConfig(cfg)))
	e.GET(
		"/internal/songs",
		func(c *echo.Context) error {
			return c.NoContent(http.StatusOK)
		},
		appmiddleware.FirebaseMaintenanceMiddleware(maintenance, nil),
	)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/songs", nil)
	req.Header.Set(echo.HeaderOrigin, origin)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, origin, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	assert.Contains(t, rec.Header().Get(echo.HeaderAccessControlExposeHeaders), echo.HeaderRetryAfter)
	assert.Equal(t, "60", rec.Header().Get(echo.HeaderRetryAfter))
}

func TestMaintenanceResponse_一時保存APIの追加オリジンへCORSヘッダーを返す(t *testing.T) {
	// Given
	cfg := config.Config{
		CORS: config.CORS{
			AllowOrigins: []string{"https://chunisupport.example.com"},
		},
	}
	maintenance := stubMaintenanceUsecase{state: usecase.MaintenanceState{Enabled: true}}
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	e.Use(echoMiddleware.CORSWithConfig(newDefaultCORSConfig(cfg)))
	registerRoutes(
		e,
		newAuthorizationTestHandlers(),
		stubFirebaseAuthenticator{},
		stubFirebaseAuthenticator{},
		stubAPITokenUsecase{},
		maintenance,
		cfg,
	)
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/internal/player-data/temp",
		nil,
	)
	req.Header.Set(echo.HeaderOrigin, info.ExternalCORSAllowOrigin)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, info.ExternalCORSAllowOrigin, rec.Header().Get(echo.HeaderAccessControlAllowOrigin))
	assert.Contains(t, rec.Header().Get(echo.HeaderAccessControlExposeHeaders), echo.HeaderRetryAfter)
	assert.Equal(t, "60", rec.Header().Get(echo.HeaderRetryAfter))
}

func TestHandleExternalHealth_外部監視向けに204NoContentを返す(t *testing.T) {
	// Given
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := handleExternalHealth(c)

	// Then
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestHandleRoot_公開情報としてビルド日だけを返す(t *testing.T) {
	// Given
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := handleRoot(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, info.Name, response["app_name"])
	assert.Equal(t, info.BuildDate, response["build_date"])
	assert.NotContains(t, response, "revision")
}

func TestHandleAdminBuildInfo_APIコミットハッシュを返す(t *testing.T) {
	// Given
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/internal/admin/build-info", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := handleAdminBuildInfo(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, info.Name, response["app_name"])
	assert.Equal(t, info.BuildDate, response["build_date"])
	assert.Equal(t, info.Revision, response["commit_hash"])
	assert.Equal(t, runtime.Version(), response["go_version"])
}

func TestHandleVersion_APIバージョン識別子を返す(t *testing.T) {
	// Given
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// When
	err := handleVersion(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.Equal(t, info.Name, response["app_name"])
	assert.Equal(t, info.BuildDate, response["build_date"])
	assert.Equal(t, info.Revision, response["commit_hash"])
	assert.Equal(t, runtime.Version(), response["go_version"])
}

func TestRequireRecentSignInVerifier(t *testing.T) {
	tests := []struct {
		name      string
		verifier  usecase.TokenVerifier
		wantNil   bool
		wantPanic string
	}{
		{
			name:     "nil のときは nil を返す",
			verifier: nil,
			wantNil:  true,
		},
		{
			name:     "RecentSignInVerifier を実装しているときはそのまま返す",
			verifier: tokenVerifierWithRecentSignIn{},
			wantNil:  false,
		},
		{
			name:      "RecentSignInVerifier を実装していないときは panic する",
			verifier:  tokenVerifierWithoutRecentSignIn{},
			wantPanic: "firebase token verifier must implement recent sign-in verifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic != "" {
				var recovered any
				assert.Panics(t, func() {
					requireRecentSignInVerifier(tt.verifier)
				})
				func() {
					defer func() {
						recovered = recover()
					}()
					requireRecentSignInVerifier(tt.verifier)
				}()
				panicMessage, ok := recovered.(string)
				assert.True(t, ok)
				assert.Contains(t, panicMessage, tt.wantPanic)
				assert.Contains(t, panicMessage, "tokenVerifierWithoutRecentSignIn")
				return
			}

			got := requireRecentSignInVerifier(tt.verifier)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}

			assert.NotNil(t, got)
		})
	}
}

type tokenVerifierWithRecentSignIn struct{}

func (tokenVerifierWithRecentSignIn) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	return "", nil
}

func (tokenVerifierWithRecentSignIn) VerifyRecentSignIn(ctx context.Context, idToken string) (*usecase.RecentSignInInfo, error) {
	return nil, nil
}

type tokenVerifierWithoutRecentSignIn struct{}

func (tokenVerifierWithoutRecentSignIn) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	return "", nil
}
