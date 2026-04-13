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
	dto "github.com/chunisupport/chunisupport-api/internal/dto"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func (m *mockUserService) GetUserProfile(ctx context.Context, username string, requester *entity.User) (*dto_internal.UserProfileDTO, error) {
	args := m.Called(ctx, username, requester)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto_internal.UserProfileDTO), args.Error(1)
}

func TestUserHandler_GetUserProfile(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockUserService)
	h := api_internal.NewUserHandler(mockService)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)

	t.Run("正常系: username と player を返す", func(t *testing.T) {
		result := &dto_internal.UserProfileDTO{
			Username: "testuser",
			Player: &dto.PlayerDTO{
				Name:      "テストプレイヤー",
				Level:     100,
				UpdatedAt: now,
				Honors:    []*dto.HonorDTO{},
			},
		}
		mockService.On("GetUserProfile", mock.Anything, "testuser", (*entity.User)(nil)).Return(result, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/profile", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("testuser")

		err := h.GetUserProfile(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var body dto_internal.UserProfileDTO
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "testuser", body.Username)
		assert.NotNil(t, body.Player)
		assert.Equal(t, "テストプレイヤー", body.Player.Name)
		mockService.AssertExpectations(t)
	})

	t.Run("正常系: プレイヤー未連携なら player は null を返す", func(t *testing.T) {
		result := &dto_internal.UserProfileDTO{
			Username: "testuser",
			Player:   nil,
		}
		mockService.On("GetUserProfile", mock.Anything, "testuser", (*entity.User)(nil)).Return(result, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser/profile", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("testuser")

		err := h.GetUserProfile(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, "testuser", body["username"])
		assert.Nil(t, body["player"])
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
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				mockService.On("GetUserProfile", mock.Anything, "testuser", (*entity.User)(nil)).Return((*dto_internal.UserProfileDTO)(nil), testCase.usecaseError).Once()

				req := httptest.NewRequest(http.MethodGet, "/users/testuser/profile", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("username")
				c.SetParamValues("testuser")

				err := h.GetUserProfile(c)

				assert.ErrorIs(t, err, testCase.expectedError)
				mockService.AssertExpectations(t)
			})
		}
	})
}
