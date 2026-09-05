package api_internal

import (
	"context"
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

type stubInternalScoreHistoryUsecase struct {
	getStandard  func(context.Context, string, *entity.User, string, string) ([]usecase.ScoreHistoryEntry, error)
	getWorldsend func(context.Context, string, *entity.User, string) ([]usecase.ScoreHistoryEntry, error)
}

func (s *stubInternalScoreHistoryUsecase) GetStandard(ctx context.Context, username string, requester *entity.User, displayID, difficulty string) ([]usecase.ScoreHistoryEntry, error) {
	return s.getStandard(ctx, username, requester, displayID, difficulty)
}

func (s *stubInternalScoreHistoryUsecase) GetWorldsend(ctx context.Context, username string, requester *entity.User, displayID string) ([]usecase.ScoreHistoryEntry, error) {
	return s.getWorldsend(ctx, username, requester, displayID)
}

func TestInternalScoreHistoryHandler_GetStandard(t *testing.T) {
	t.Run("Firebase認証ユーザーを閲覧者として渡す", func(t *testing.T) {
		requester := &entity.User{ID: 1}
		h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{
			getStandard: func(_ context.Context, username string, gotRequester *entity.User, displayID, difficulty string) ([]usecase.ScoreHistoryEntry, error) {
				assert.Equal(t, "testuser", username)
				assert.Same(t, requester, gotRequester)
				assert.Equal(t, "0123456789abcdef", displayID)
				assert.Equal(t, "MASTER", difficulty)
				return []usecase.ScoreHistoryEntry{}, nil
			},
		})
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/songs/song/master/history", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/internal/users/:username/record/songs/:displayid/:difficulty/history")
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "displayid", Value: "0123456789abcdef"}, {Name: "difficulty", Value: "master"}})
		c.Set("userEntity", requester)

		err := h.GetStandard(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("不正なユーザー名は400を返す", func(t *testing.T) {
		h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/internal/users/-/record/songs/song/master/history", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "-"}, {Name: "displayid", Value: "0123456789abcdef"}, {Name: "difficulty", Value: "master"}})

		err := h.GetStandard(c)

		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	})

	t.Run("不正な難易度は400を返す", func(t *testing.T) {
		h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/songs/song/invalid/history", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "displayid", Value: "0123456789abcdef"}, {Name: "difficulty", Value: "invalid"}})

		err := h.GetStandard(c)

		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeInvalidDifficulty, apiErr.Code)
	})
}

func TestInternalScoreHistoryHandler_GetWorldsend(t *testing.T) {
	requester := &entity.User{ID: 1}
	h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{
		getWorldsend: func(_ context.Context, username string, gotRequester *entity.User, displayID string) ([]usecase.ScoreHistoryEntry, error) {
			assert.Equal(t, "testuser", username)
			assert.Same(t, requester, gotRequester)
			assert.Equal(t, "0123456789abcdef", displayID)
			return []usecase.ScoreHistoryEntry{}, nil
		},
	})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/worldsend-songs/song/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/internal/users/:username/record/worldsend-songs/:displayid/history")
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "displayid", Value: "0123456789abcdef"}})
	c.Set("userEntity", requester)

	err := h.GetWorldsend(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestInternalScoreHistoryHandler_GetStandard_不正な楽曲IDはUsecaseへ渡さず400系エラーを返す(t *testing.T) {
	h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/songs/invalid/master/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "displayid", Value: "invalid"}, {Name: "difficulty", Value: "master"}})

	err := h.GetStandard(c)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
}

func TestInternalScoreHistoryHandler_GetWorldsend_不正な楽曲IDはUsecaseへ渡さず400系エラーを返す(t *testing.T) {
	h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/worldsend-songs/invalid/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}, {Name: "displayid", Value: "invalid"}})

	err := h.GetWorldsend(c)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnprocessableEntity, apiErr.HTTPStatus)
	assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
}

func TestInternalScoreHistoryHandler_GetWorldsend_不正なユーザー名は400を返す(t *testing.T) {
	h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/-/record/worldsend-songs/song/history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "-"}, {Name: "displayid", Value: "0123456789abcdef"}})

	err := h.GetWorldsend(c)

	apiErr, ok := err.(*apierror.APIError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
}
