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
	"github.com/chunisupport/chunisupport-api/internal/dto"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockUserService は usecase.UserUsecase のモックです。
type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*dto_internal.UserProfileWithRecordsDTO, error) {
	args := m.Called(ctx, username, requester, includeNoPlay)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto_internal.UserProfileWithRecordsDTO), args.Error(1)
}

func (m *mockUserService) GetUserProfileRatingView(ctx context.Context, username string, requester *entity.User) (*dto_internal.UserProfileRatingViewDTO, error) {
	args := m.Called(ctx, username, requester)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto_internal.UserProfileRatingViewDTO), args.Error(1)
}

func (m *mockUserService) GetUserProfileRecordView(ctx context.Context, username string, requester *entity.User, includeNoPlay bool) (*dto_internal.UserProfileRecordViewDTO, error) {
	args := m.Called(ctx, username, requester, includeNoPlay)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto_internal.UserProfileRecordViewDTO), args.Error(1)
}

func (m *mockUserService) GetAllUsersForAdmin(ctx context.Context, page int, limit int, name string) ([]dto_internal.AdminUserListResponse, error) {
	args := m.Called(ctx, page, limit, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto_internal.AdminUserListResponse), args.Error(1)
}

func (m *mockUserService) DeleteUser(ctx context.Context, requester *entity.User, username string) error {
	args := m.Called(ctx, requester, username)
	return args.Error(0)
}

func TestUserHandler_GetUserProfileWithRecords(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockUserService)
	h := api_internal.NewUserHandler(mockService)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	player := &dto.PlayerDTO{
		Name:      "player",
		Level:     10,
		Honors:    []*dto.HonorDTO{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	records := &dto.UserRecordResponseDTO{
		UpdatedAt:     now,
		Best:          []*dto.PlayerRecordDTO{{ID: "best1"}},
		BestCandidate: []*dto.PlayerRecordDTO{{ID: "best_candidate1"}},
		New:           []*dto.PlayerRecordDTO{{ID: "new1"}},
		NewCandidate:  []*dto.PlayerRecordDTO{{ID: "new_candidate1"}},
		All:           []*dto.PlayerRecordDTO{{ID: "all1"}},
		WorldsEnd:     []*dto.WorldsendRecordDTO{{ID: "we1"}},
	}
	result := &dto_internal.UserProfileWithRecordsDTO{
		Username:  "testuser",
		Player:    player,
		Records:   records,
		UpdatedAt: &now,
	}
	ratingRecords := &dto_internal.UserRatingRecordResponseDTO{
		UpdatedAt:     now,
		Best:          []*dto.PlayerRecordDTO{{ID: "best1"}},
		BestCandidate: []*dto.PlayerRecordDTO{{ID: "best_candidate1"}},
		New:           []*dto.PlayerRecordDTO{{ID: "new1"}},
		NewCandidate:  []*dto.PlayerRecordDTO{{ID: "new_candidate1"}},
	}
	ratingResult := &dto_internal.UserProfileRatingViewDTO{
		Username:  "testuser",
		Player:    player,
		Records:   ratingRecords,
		UpdatedAt: &now,
	}

	t.Run("viewなしは全レコードを返す", func(t *testing.T) {
		mockService.On("GetUserProfileWithRecords", mock.Anything, "testuser", (*entity.User)(nil), false).Return(result, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("testuser")

		err := h.GetUserProfileWithRecords(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		recordsBody, ok := body["records"].(map[string]any)
		assert.True(t, ok)
		_, hasAll := recordsBody["all"]
		_, hasWorldsend := recordsBody["worldsend"]
		assert.True(t, hasAll)
		assert.True(t, hasWorldsend)
		mockService.AssertExpectations(t)
	})

	t.Run("view=ratingはレーティング枠のみ返す", func(t *testing.T) {
		mockService.On("GetUserProfileRatingView", mock.Anything, "testuser", (*entity.User)(nil)).Return(ratingResult, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/users/testuser?view=rating", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("testuser")

		err := h.GetUserProfileWithRecords(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		recordsBody, ok := body["records"].(map[string]any)
		assert.True(t, ok)
		_, hasAll := recordsBody["all"]
		_, hasWorldsend := recordsBody["worldsend"]
		assert.False(t, hasAll)
		assert.False(t, hasWorldsend)
		mockService.AssertExpectations(t)
	})

	t.Run("view=ratingの異常系", func(t *testing.T) {
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
				name:          "プレイヤー未紐付け",
				usecaseError:  usecase.ErrPlayerNotLinked,
				expectedError: apierror.ErrUserNotFound,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				mockService.On("GetUserProfileRatingView", mock.Anything, "testuser", (*entity.User)(nil)).Return((*dto_internal.UserProfileRatingViewDTO)(nil), testCase.usecaseError).Once()

				req := httptest.NewRequest(http.MethodGet, "/users/testuser?view=rating", nil)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				c.SetParamNames("username")
				c.SetParamValues("testuser")

				err := h.GetUserProfileWithRecords(c)

				assert.ErrorIs(t, err, testCase.expectedError)
				mockService.AssertExpectations(t)
			})
		}
	})
}

func TestUserHandler_GetUserProfileWithRecordView(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockUserService)
	h := api_internal.NewUserHandler(mockService)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	player := &dto.PlayerDTO{
		Name:      "player",
		Level:     10,
		Honors:    []*dto.HonorDTO{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	recordViewResult := &dto_internal.UserProfileRecordViewDTO{
		Username: "testuser",
		Player:   player,
		Records: &dto_internal.UserRecordViewResponseDTO{
			UpdatedAt: now,
			All:       []*dto.PlayerRecordDTO{{ID: "all1"}},
			Worldsend: []*dto.WorldsendRecordDTO{{ID: "we1"}},
		},
		UpdatedAt: &now,
	}

	mockService.On("GetUserProfileRecordView", mock.Anything, "testuser", (*entity.User)(nil), true).Return(recordViewResult, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/users/testuser?view=record&include_noplay=true", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("username")
	c.SetParamValues("testuser")

	err := h.GetUserProfileWithRecords(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	recordsBody, ok := body["records"].(map[string]any)
	assert.True(t, ok)
	_, hasAll := recordsBody["all"]
	_, hasWorldsend := recordsBody["worldsend"]
	_, hasBest := recordsBody["best"]
	_, hasNew := recordsBody["new"]
	assert.True(t, hasAll)
	assert.True(t, hasWorldsend)
	assert.False(t, hasBest)
	assert.False(t, hasNew)
	mockService.AssertExpectations(t)
}

func TestAdminUserHandler_GetAllUsers(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockUserService)
	h := api_internal.NewAdminUserHandler(mockService)
	createdAt := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)

	expected := []dto_internal.AdminUserListResponse{
		{
			UserName:       "user1",
			AccountType:    "EDITOR",
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			PlayerName:     new("player1"),
			Rating:         new(17.25),
			OverPowerValue: new(float64(9500)),
			IsSuspicious:   true,
			IsPrivate:      false,
		},
		{
			UserName:       "user2",
			AccountType:    "PLAYER",
			CreatedAt:      createdAt.Add(time.Hour),
			UpdatedAt:      updatedAt.Add(time.Hour),
			PlayerName:     nil,
			Rating:         nil,
			OverPowerValue: nil,
			IsSuspicious:   false,
			IsPrivate:      true,
		},
	}

	mockService.On("GetAllUsersForAdmin", mock.Anything, 2, info.DefaultUserListLimit, "user").Return(expected, nil).Once()

	req := httptest.NewRequest(http.MethodGet, "/internal/users?page=2&name=user", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.GetAllUsers(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var body []map[string]any
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body, 2)
	assert.Equal(t, "user1", body[0]["username"])
	assert.Equal(t, "EDITOR", body[0]["account_type"])
	assert.Equal(t, createdAt.Format(time.RFC3339), body[0]["created_at"])
	assert.Equal(t, updatedAt.Format(time.RFC3339), body[0]["updated_at"])
	assert.Equal(t, "player1", body[0]["player_name"])
	assert.Equal(t, 17.25, body[0]["rating"])
	assert.Equal(t, 9500.0, body[0]["overpower_value"])
	assert.Equal(t, true, body[0]["is_suspicious"])
	assert.Equal(t, false, body[0]["is_private"])
	assert.Equal(t, "user2", body[1]["username"])
	assert.Equal(t, "PLAYER", body[1]["account_type"])
	assert.Equal(t, createdAt.Add(time.Hour).Format(time.RFC3339), body[1]["created_at"])
	assert.Equal(t, updatedAt.Add(time.Hour).Format(time.RFC3339), body[1]["updated_at"])
	assert.Nil(t, body[1]["player_name"])
	assert.Nil(t, body[1]["rating"])
	assert.Nil(t, body[1]["overpower_value"])
	assert.Equal(t, false, body[1]["is_suspicious"])
	assert.Equal(t, true, body[1]["is_private"])
	mockService.AssertExpectations(t)
}

func TestUserHandler_DeleteUser(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockUserService)
	h := api_internal.NewUserHandler(mockService)
	adminUser := &entity.User{ID: 99, AccountTypeID: 3}

	t.Run("正常系: ユーザー削除", func(t *testing.T) {
		mockService.On("DeleteUser", mock.Anything, adminUser, "testuser").Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/testuser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("testuser")
		c.Set("userEntity", adminUser)

		err := h.DeleteUser(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("異常系: ユーザーが存在しない", func(t *testing.T) {
		mockService.On("DeleteUser", mock.Anything, adminUser, "nonexistent").Return(usecase.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/nonexistent", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("nonexistent")
		c.Set("userEntity", adminUser)

		err := h.DeleteUser(c)

		assert.Error(t, err)
		mockService.AssertExpectations(t)
	})

	t.Run("異常系: ADMIN権限がない", func(t *testing.T) {
		normalUser := &entity.User{ID: 1, AccountTypeID: 1}
		mockService.On("DeleteUser", mock.Anything, normalUser, "testuser").Return(usecase.ErrAdminRequired).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/testuser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("testuser")
		c.Set("userEntity", normalUser)

		err := h.DeleteUser(c)

		assert.ErrorIs(t, err, apierror.ErrForbidden)
		mockService.AssertExpectations(t)
	})
}
