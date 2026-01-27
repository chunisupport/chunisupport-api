package api_internal_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Qman110101/chunisupport-api/internal/app/handler/api_internal"
	"github.com/Qman110101/chunisupport-api/internal/domain/entity"
	"github.com/Qman110101/chunisupport-api/internal/dto"
	dto_internal "github.com/Qman110101/chunisupport-api/internal/dto/api_internal"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockUserService は usecase.UserUsecase のモックです。
type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) GetUserProfileWithRecords(ctx context.Context, username string, requester *entity.User) (*dto_internal.UserProfileWithRecordsDTO, error) {
	args := m.Called(ctx, username, requester)
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

func (m *mockUserService) GetAllUsersForAdmin(ctx context.Context, page int, limit int, name string) ([]dto_internal.AdminUserListResponse, error) {
	args := m.Called(ctx, page, limit, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]dto_internal.AdminUserListResponse), args.Error(1)
}

func (m *mockUserService) DeleteUser(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}

func (m *mockUserService) RestoreUser(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
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
		mockService.On("GetUserProfileWithRecords", mock.Anything, "testuser", (*entity.User)(nil)).Return(result, nil).Once()

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
}

func TestUserHandler_DeleteUser(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockUserService)
	h := api_internal.NewUserHandler(mockService)

	t.Run("正常系: ユーザー削除", func(t *testing.T) {
		mockService.On("DeleteUser", mock.Anything, "testuser").Return(nil).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/testuser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("testuser")

		err := h.DeleteUser(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("異常系: ユーザーが存在しない", func(t *testing.T) {
		mockService.On("DeleteUser", mock.Anything, "nonexistent").Return(usecase.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/nonexistent", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("nonexistent")

		err := h.DeleteUser(c)

		assert.Error(t, err)
		mockService.AssertExpectations(t)
	})

	t.Run("異常系: 既に削除済み", func(t *testing.T) {
		mockService.On("DeleteUser", mock.Anything, "deleteduser").Return(usecase.ErrUserAlreadyDeleted).Once()

		req := httptest.NewRequest(http.MethodDelete, "/users/deleteduser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("deleteduser")

		err := h.DeleteUser(c)

		assert.Error(t, err)
		mockService.AssertExpectations(t)
	})
}

func TestUserHandler_RestoreUser(t *testing.T) {
	e := newTestEcho()
	mockService := new(mockUserService)
	h := api_internal.NewUserHandler(mockService)

	t.Run("正常系: ユーザー復活", func(t *testing.T) {
		mockService.On("RestoreUser", mock.Anything, "deleteduser").Return(nil).Once()

		req := httptest.NewRequest(http.MethodPost, "/users/deleteduser/restore", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("deleteduser")

		err := h.RestoreUser(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, rec.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("異常系: ユーザーが存在しない", func(t *testing.T) {
		mockService.On("RestoreUser", mock.Anything, "nonexistent").Return(usecase.ErrUserNotFound).Once()

		req := httptest.NewRequest(http.MethodPost, "/users/nonexistent/restore", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("nonexistent")

		err := h.RestoreUser(c)

		assert.Error(t, err)
		mockService.AssertExpectations(t)
	})

	t.Run("異常系: 削除されていない", func(t *testing.T) {
		mockService.On("RestoreUser", mock.Anything, "activeuser").Return(usecase.ErrUserNotDeleted).Once()

		req := httptest.NewRequest(http.MethodPost, "/users/activeuser/restore", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("username")
		c.SetParamValues("activeuser")

		err := h.RestoreUser(c)

		assert.Error(t, err)
		mockService.AssertExpectations(t)
	})
}
