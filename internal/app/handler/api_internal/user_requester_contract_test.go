package api_internal_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserHandler_閲覧者をusecaseに渡す(t *testing.T) {
	for _, tt := range []struct {
		name, method, path string
		invoke             func(*api_internal.UserHandler, *echo.Context) error
		extra              []any
	}{
		{"プロフィール", "GetUserProfile", "/", (*api_internal.UserHandler).GetUserProfile, nil},
		{"更新日時", "GetUserUpdatedAt", "/", (*api_internal.UserHandler).GetUserUpdatedAt, nil},
		{"レート", "GetUserProfileRatingView", "/", (*api_internal.UserHandler).GetUserRating, nil},
		{"レコード", "GetUserProfileRecordView", "/", (*api_internal.UserHandler).GetUserRecord, nil},
		{"一括", "GetUserProfileWithRecords", "/", (*api_internal.UserHandler).GetUserProfileWithRecords, nil},
		{"一括レート", "GetUserProfileRatingView", "/?view=rating", (*api_internal.UserHandler).GetUserProfileWithRecords, nil},
		{"一括レコード", "GetUserProfileRecordView", "/?view=record", (*api_internal.UserHandler).GetUserProfileWithRecords, nil},
		{"通常曲", "GetUserSongRecord", "/?difficulty=MASTER", (*api_internal.UserHandler).GetUserSongRecord, []any{"0000000000000001", "MASTER"}},
		{"WE曲", "GetUserWorldsendSongRecord", "/", (*api_internal.UserHandler).GetUserWorldsendSongRecord, []any{"0000000000000001"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requester := &entity.User{ID: 7}
			uc := new(mockUserUsecase)
			args := append([]any{mock.Anything, "targetuser", requester}, tt.extra...)
			// 正常応答のDTO検証とは分け、認可判断に必要な閲覧者の配線だけを保証する。
			uc.On(tt.method, args...).Return(nil, usecase.ErrUserNotFound).Once()
			e := newTestEcho()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, tt.path, nil), httptest.NewRecorder())
			c.SetPathValues(echo.PathValues{{Name: "username", Value: "targetuser"}, {Name: "id", Value: "0000000000000001"}})
			c.Set("userEntity", requester)
			err := tt.invoke(api_internal.NewUserHandler(uc), c)
			assert.ErrorIs(t, err, apierror.ErrUserNotFound)
			uc.AssertExpectations(t)
		})
	}
}
