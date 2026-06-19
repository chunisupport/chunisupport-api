package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
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
				e.OPTIONS("/healthz", func(c echo.Context) error {
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
				e.OPTIONS("/", func(c echo.Context) error {
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
				e.OPTIONS("/internal/player-data/temp", func(c echo.Context) error {
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
				e.OPTIONS("/internal/users/:username", func(c echo.Context) error {
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
