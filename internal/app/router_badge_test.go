package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	internalhandler "github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type badgeUserUsecaseStub struct {
	usecase.UserUsecase
	rating *float64
	err    error
}

func (s badgeUserUsecaseStub) GetPublicOfficialRating(context.Context, string) (*float64, error) {
	return s.rating, s.err
}

func TestOfficialRatingBadgeRoute_匿名取得と非公開の遮断(t *testing.T) {
	rating := 17.29
	tests := []struct {
		name       string
		usecase    badgeUserUsecaseStub
		wantStatus int
		wantBody   string
	}{
		{name: "認証なしで公式RATINGを取得できる", usecase: badgeUserUsecaseStub{rating: &rating}, wantStatus: http.StatusOK, wantBody: `"message":"17.29"`},
		{name: "非公開ユーザーは404になる", usecase: badgeUserUsecaseStub{err: usecase.ErrUserPrivate}, wantStatus: http.StatusNotFound, wantBody: `"code":"user_not_found"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			handlers := newAuthorizationTestHandlers()
			handlers.User = internalhandler.NewUserHandler(tt.usecase)
			registerRoutes(e, handlers, stubFirebaseAuthenticator{}, stubFirebaseAuthenticator{}, stubAPITokenUsecase{}, stubMaintenanceUsecase{}, config.Config{})
			req := httptest.NewRequest(http.MethodGet, "/badges/users/testuser/rating", nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), tt.wantBody)
		})
	}
}
