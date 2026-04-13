package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
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
			name:          "ルートでは追加オリジンを許可する",
			path:          "/",
			requestMethod: http.MethodGet,
			setupRoute: func(e *echo.Echo) {
				rootCORS := echoMiddleware.CORSWithConfig(newExternalCORSConfig(cfg))
				e.OPTIONS("/", func(c echo.Context) error {
					return c.NoContent(http.StatusNoContent)
				}, rootCORS)
			},
			wantAllow:     "https://new.chunithm-net.com",
			wantAllowCred: "true",
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
