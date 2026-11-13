package api_v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockV1UserUsecase struct {
	getUserProfileWithRecordsFunc func(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileWithRecordsOutput, error)
	getUserProfileRatingViewFunc  func(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileRatingViewOutput, error)
}

func (m *mockV1UserUsecase) GetPublicOfficialRating(context.Context, string) (*float64, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetPublicBadgePlayer(context.Context, string) (*entity.Player, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserProfile(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileOutput, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserUpdatedAt(ctx context.Context, username string, requester *entity.User) (*usecase.UserUpdatedAtOutput, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileWithRecordsOutput, error) {
	if m.getUserProfileWithRecordsFunc != nil {
		return m.getUserProfileWithRecordsFunc(ctx, username, requester)
	}
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserProfileRatingView(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileRatingViewOutput, error) {
	if m.getUserProfileRatingViewFunc != nil {
		return m.getUserProfileRatingViewFunc(ctx, username, requester)
	}
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserProfileRecordView(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileRecordViewOutput, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserSongRecord(ctx context.Context, username string, requester *entity.User, displayID string, difficulty string) (*usecase.UserSongRecordOutput, error) {
	return nil, nil
}

func (m *mockV1UserUsecase) GetUserWorldsendSongRecord(ctx context.Context, username string, requester *entity.User, displayID string) (*usecase.UserWorldsendSongRecordOutput, error) {
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
			getUserProfileWithRecordsFunc: func(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileWithRecordsOutput, error) {
				assert.Equal(t, "privateuser", username)
				return nil, usecase.ErrUserPrivate
			},
		}
		handler := NewV1UserHandler(mockUsecase)
		req := httptest.NewRequest(http.MethodGet, "/v1/users/privateuser", nil)
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
			getUserProfileWithRecordsFunc: func(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileWithRecordsOutput, error) {
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
		updatedAt := time.Date(2026, time.August, 27, 12, 34, 56, 0, time.UTC)
		recordUpdatedAt := time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)
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
					Records: &usecase.UserRatingRecordOutput{
						UpdatedAt: updatedAt,
						Best: []*usecase.PlayerRecordOutput{{
							PlayerRecord: &entity.PlayerRecord{
								Song: &entity.Song{DisplayID: "0000000000000001"},
							},
							UpdatedAt:  &recordUpdatedAt,
							IsPlayed:   true,
							Difficulty: "MASTER",
						}},
					},
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
		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		// usecase の結果が変換されて返ることを代表1件で確かめる（変換の詳細はDTOテストで固定する）。
		assert.JSONEq(t, `17.1234`, string(body["rating"]))
		assert.JSONEq(t, `17.2345`, string(body["best_average"]))
		assert.JSONEq(t, `16.9567`, string(body["new_average"]))
		var best []map[string]any
		require.NoError(t, json.Unmarshal(body["best"], &best))
		if assert.Len(t, best, 1) {
			assert.Equal(t, "0000000000000001", best[0]["id"])
			assert.Equal(t, "MASTER", best[0]["difficulty"])
			assert.Equal(t, recordUpdatedAt.Format(time.RFC3339), best[0]["updated_at"])
			assert.Equal(t, true, best[0]["is_played"])
		}
		// 残りの3枠は空配列を返す（null・キー欠落と区別する）。
		assert.JSONEq(t, `[]`, string(body["best_candidate"]))
		assert.JSONEq(t, `[]`, string(body["new"]))
		assert.JSONEq(t, `[]`, string(body["new_candidate"]))
		assert.JSONEq(t, `{"updated_at":"2026-08-27T12:34:56Z"}`, string(body["meta"]))
		// レーティング専用APIは standard・worldsend・course を含まない。
		for _, key := range []string{"standard", "worldsend", "course"} {
			_, ok := body[key]
			assert.False(t, ok, "%s を含んではならない", key)
		}
	})

	t.Run("認証ユーザーを閲覧者として渡す", func(t *testing.T) {
		// Given
		e := echo.New()
		requester := &entity.User{ID: 10}
		var gotUsername string
		var gotRequester *entity.User
		called := false
		mockUsecase := &mockV1UserUsecase{
			getUserProfileRatingViewFunc: func(ctx context.Context, username string, actualRequester *entity.User) (*usecase.UserProfileRatingViewOutput, error) {
				called = true
				gotUsername = username
				gotRequester = actualRequester
				return &usecase.UserProfileRatingViewOutput{}, nil
			},
		}
		handler := NewV1UserHandler(mockUsecase)
		req := httptest.NewRequest(http.MethodGet, "/v1/users/testuser/rating", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})
		c.Set("userEntity", requester)

		// When
		err := handler.GetUserRating(c)

		// Then
		require.NoError(t, err)
		// usecase が呼ばれ、認証ユーザーが閲覧者として渡されたことを外側で確かめる。
		assert.True(t, called)
		assert.Equal(t, "testuser", gotUsername)
		assert.Same(t, requester, gotRequester)
	})

	t.Run("非公開ユーザーはuser_not_foundを返す", func(t *testing.T) {
		// Given
		e := echo.New()
		mockUsecase := &mockV1UserUsecase{
			getUserProfileRatingViewFunc: func(ctx context.Context, username string, requester *entity.User) (*usecase.UserProfileRatingViewOutput, error) {
				return nil, usecase.ErrUserPrivate
			},
		}
		handler := NewV1UserHandler(mockUsecase)
		req := httptest.NewRequest(http.MethodGet, "/v1/users/privateuser/rating", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "privateuser"}})

		// When
		err := handler.GetUserRating(c)

		// Then
		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeUserNotFound, apiErr.Code)
			assert.Equal(t, http.StatusNotFound, apiErr.HTTPStatus)
		}
	})
}

var _ usecase.UserUsecase = (*mockV1UserUsecase)(nil)
