package api_internal_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserHandler_CancellationLogging(t *testing.T) {
	endpoints := []struct {
		name    string
		method  string
		handler func(*api_internal.UserHandler, *echo.Context) error
	}{
		{"プロフィール", "GetUserProfile", (*api_internal.UserHandler).GetUserProfile},
		{"更新日時", "GetUserUpdatedAt", (*api_internal.UserHandler).GetUserUpdatedAt},
		{"レーティング", "GetUserProfileRatingView", (*api_internal.UserHandler).GetUserRating},
		{"レコード", "GetUserProfileRecordView", (*api_internal.UserHandler).GetUserRecord},
		{"一括取得", "GetUserProfileWithRecords", (*api_internal.UserHandler).GetUserProfileWithRecords},
	}
	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			for _, tt := range []struct {
				name   string
				cause  error
				status int
			}{
				{"キャンセル", context.Canceled, info.StatusClientClosedRequest},
				{"期限超過", context.DeadlineExceeded, http.StatusInternalServerError},
				{"通常の失敗", errors.New("database unavailable"), http.StatusInternalServerError},
			} {
				t.Run(tt.name, func(t *testing.T) {
					var output bytes.Buffer
					original := slog.Default()
					slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
					t.Cleanup(func() { slog.SetDefault(original) })
					uc := new(mockUserUsecase)
					wrapped := errors.Join(errors.New("repository operation failed"), tt.cause)
					uc.On(endpoint.method, mock.Anything, "tester", mock.Anything).Return(nil, wrapped).Once()
					e := echo.New()
					rec := httptest.NewRecorder()
					c := e.NewContext(httptest.NewRequest(http.MethodGet, "/users/tester", nil), rec)
					c.SetPathValues(echo.PathValues{{Name: "username", Value: "tester"}})

					err := endpoint.handler(api_internal.NewUserHandler(uc), c)

					require.ErrorIs(t, err, tt.cause)
					middleware.CustomHTTPErrorHandler(c, err)
					assert.Equal(t, tt.status, rec.Code)
					if tt.status == info.StatusClientClosedRequest {
						assert.Equal(t, 1, strings.Count(output.String(), "level=INFO"))
						assert.NotContains(t, output.String(), "level=WARN")
						assert.NotContains(t, output.String(), "level=ERROR")
						assert.Empty(t, rec.Body.String())
					} else {
						assert.Contains(t, output.String(), "level=ERROR")
						assert.NotContains(t, output.String(), "canceled by client")
					}
					uc.AssertExpectations(t)
				})
			}
		})
	}
}
