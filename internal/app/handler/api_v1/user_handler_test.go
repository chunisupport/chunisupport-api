package api_v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockV1UserUsecase struct {
	getUserProfileWithRecordsFunc func(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*usecase.UserProfileWithRecordsOutput, error)
	getUserProfileRatingViewFunc  func(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileRatingViewOutput, error)
}

func (m *mockV1UserUsecase) GetUserProfile(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileOutput, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserUpdatedAt(ctx context.Context, username string, requester *entity.User) (*usecase.UserUpdatedAtOutput, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*usecase.UserProfileWithRecordsOutput, error) {
	if m.getUserProfileWithRecordsFunc != nil {
		return m.getUserProfileWithRecordsFunc(ctx, username, requester, includeNoPlay)
	}
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserProfileRatingView(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileRatingViewOutput, error) {
	if m.getUserProfileRatingViewFunc != nil {
		return m.getUserProfileRatingViewFunc(ctx, username, requester)
	}
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserProfileRecordView(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*usecase.UserProfileRecordViewOutput, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserSongRecord(ctx context.Context, username string, requester *entity.User, displayID string, includeNoPlay bool, difficulty string) (*usecase.UserSongRecordOutput, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserWorldsendSongRecord(ctx context.Context, username string, requester *entity.User, displayID string, includeNoPlay bool) (*usecase.UserWorldsendSongRecordOutput, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetAllUsersForAdmin(ctx context.Context, page int, limit int, name string) ([]usecase.AdminUserOutput, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) DeleteUser(ctx context.Context, requester *entity.User, username string) error {
	return nil
}

func TestV1UserHandler_GetUser(t *testing.T) {
	t.Run("非公開ユーザーはuser_not_foundを返す", func(t *testing.T) {
		// Given
		e := echo.New()
		mockUsecase := &mockV1UserUsecase{
			getUserProfileWithRecordsFunc: func(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*usecase.UserProfileWithRecordsOutput, error) {
				assert.Equal(t, "privateuser", username)
				assert.True(t, includeNoPlay)
				return nil, usecase.ErrUserPrivate
			},
		}
		handler := NewV1UserHandler(mockUsecase)
		req := httptest.NewRequest(http.MethodGet, "/v1/users/privateuser?include_noplay=true", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "privateuser"}})

		// When
		err := handler.GetUser(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeUserNotFound, apiErr.Code)
			assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
		}
	})

	t.Run("不正なusernameは境界で拒否する", func(t *testing.T) {
		// Given
		called := false
		e := echo.New()
		mockUsecase := &mockV1UserUsecase{
			getUserProfileWithRecordsFunc: func(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*usecase.UserProfileWithRecordsOutput, error) {
				called = true
				return nil, nil
			},
		}
		handler := NewV1UserHandler(mockUsecase)
		req := httptest.NewRequest(http.MethodGet, "/v1/users/PrivateUser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "PrivateUser"}})

		// When
		err := handler.GetUser(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeUsernameInvalidChar, apiErr.Code)
			assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		}
		assert.False(t, called)
	})
}

func TestV1UserHandler_GetUserRating(t *testing.T) {
	t.Run("レーティング情報だけを返す", func(t *testing.T) {
		// Given
		rating := 17.1234
		bestAverage := 17.2345
		newAverage := 16.9567
		e := echo.New()
		mockUsecase := &mockV1UserUsecase{
			getUserProfileRatingViewFunc: func(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileRatingViewOutput, error) {
				assert.Equal(t, "testuser", username)
				assert.Nil(t, requester)
				return &usecase.UserProfileRatingViewOutput{
					Player: &usecase.UserPlayerOutput{Player: &entity.Player{
						CalculatedRating:  &rating,
						BestAverageRating: &bestAverage,
						NewAverageRating:  &newAverage,
					}},
					Records: &usecase.UserRatingRecordOutput{},
				}, nil
			},
		}
		handler := NewV1UserHandler(mockUsecase)
		req := httptest.NewRequest(http.MethodGet, "/v1/users/testuser/rating", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

		// When
		err := handler.GetUserRating(c)

		// Then
		require.NoError(t, err)
		var response map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, rating, response["rating"])
		assert.Equal(t, bestAverage, response["best_average"])
		assert.Equal(t, newAverage, response["new_average"])
		assert.NotContains(t, response, "standard")
		assert.NotContains(t, response, "worldsend")
		assert.NotContains(t, response, "course")
	})
}

var _ usecase.UserUsecase = (*mockV1UserUsecase)(nil)
