package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	internalhandler "github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestRegisterRoutes_認証済みユーザーの内部API参照はエンドポイントをまたいで回数を制限する(t *testing.T) {
	// Given: 通常アカウントがフレンド比較の2種類のエンドポイントを交互に呼び出す
	handlers := newAuthorizationTestHandlers()
	handlers.FriendScoreComparison = internalhandler.NewFriendScoreComparisonHandler(&stubFriendScoreComparisonUsecase{})
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, nil, stubMaintenanceUsecase{}, config.Config{})
	paths := []string{
		"/internal/friend-comparisons/frienduser/charts/MASTER",
		"/internal/friend-comparisons/frienduser/worldsend",
	}

	for i := range info.InternalAuthenticatedReadRateLimitRequests + 1 {
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, paths[i%len(paths)], nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer player-token")
		rec := httptest.NewRecorder()

		// When
		e.ServeHTTP(rec, req)

		// Then: 上限までは成功し、超過したリクエストだけ429になる
		if i < info.InternalAuthenticatedReadRateLimitRequests {
			assert.Equal(t, http.StatusOK, rec.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		}
	}
}
