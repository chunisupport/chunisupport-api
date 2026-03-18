package api_internal_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func (m *mockUserService) GetUserUpdatedAt(ctx context.Context, username string, requester *entity.User) (*dto_internal.UserUpdatedAtDTO, error) {
	args := m.Called(ctx, username, requester)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto_internal.UserUpdatedAtDTO), args.Error(1)
}

func TestUserHandler_GetUserUpdatedAt(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockUserService)
	h := api_internal.NewUserHandler(mockService)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	t.Run("正常系: updated_at を返す", func(t *testing.T) {
		result := &dto_internal.UserUpdatedAtDTO{
			UpdatedAt: now,
		}
		mockService.On("GetUserUpdatedAt", mock.Anything, "testuser", (*entity.User)(nil)).Return(result, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/updated-at", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("testuser")

		err := h.GetUserUpdatedAt(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var body dto_internal.UserUpdatedAtDTO
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.True(t, result.UpdatedAt.Equal(body.UpdatedAt))
		mockService.AssertExpectations(t)
	})

	t.Run("異常系: 既存プロフィールAPIと同じエラー変換を使う", func(t *testing.T) {
		testCases := []struct {
			name          string
			usecaseError  error
			expectedError error
		}{
			{
				name:          "ユーザーが存在しない",
				usecaseError:  usecase.ErrUserNotFound,
				expectedError: apierror.ErrUserNotFound,
			},
			{
				name:          "ユーザーが非公開",
				usecaseError:  usecase.ErrUserPrivate,
				expectedError: apierror.ErrUserNotFound,
			},
			{
				name:          "プレイヤー未連携",
				usecaseError:  usecase.ErrPlayerNotLinked,
				expectedError: apierror.ErrUserNotFound,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				mockService.On("GetUserUpdatedAt", mock.Anything, "testuser", (*entity.User)(nil)).Return((*dto_internal.UserUpdatedAtDTO)(nil), testCase.usecaseError).Once()

				req := httptest.NewRequest(http.MethodGet, "/users/testuser/updated-at", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("username")
				c.SetParamValues("testuser")

				err := h.GetUserUpdatedAt(c)

				assert.ErrorIs(t, err, testCase.expectedError)
				mockService.AssertExpectations(t)
			})
		}
	})
}
