package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPlayerPlayHistoryUsecase struct {
	get func(context.Context, string, *entity.User) ([]usecase.PlayerPlayHistoryOutput, error)
}

func (s *stubPlayerPlayHistoryUsecase) Get(ctx context.Context, username string, requester *entity.User) ([]usecase.PlayerPlayHistoryOutput, error) {
	return s.get(ctx, username, requester)
}

func TestPlayerPlayHistoryHandler_Get(t *testing.T) {
	// Given
	requester := &entity.User{ID: 1}
	title := "解決済み楽曲"
	difficulty := "MASTER"
	clearLamp := "CLEAR"
	playedAt := time.Date(2026, time.November, 26, 12, 34, 56, 0, time.UTC)
	h := NewPlayerPlayHistoryHandler(&stubPlayerPlayHistoryUsecase{get: func(_ context.Context, username string, gotRequester *entity.User) ([]usecase.PlayerPlayHistoryOutput, error) {
		assert.Equal(t, "testuser", username)
		assert.Same(t, requester, gotRequester)
		return []usecase.PlayerPlayHistoryOutput{
			{Title: &title, Difficulty: &difficulty, Score: 1005000, ClearLamp: &clearLamp, PlayedAt: playedAt},
			{Title: nil, Difficulty: nil, Score: 500000, ClearLamp: nil, PlayedAt: playedAt.Add(-time.Minute)},
		}, nil
	}})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/record/play-history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})
	c.Set("userEntity", requester)

	// When
	err := h.Get(c)

	// Then
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "private, no-store", rec.Header().Get(echo.HeaderCacheControl))
	assert.JSONEq(t, `{"entries":[{"title":"解決済み楽曲","difficulty":"MASTER","score":1005000,"clear_lamp":"CLEAR","combo_lamp":null,"full_chain":null,"played_at":"2026-11-26T12:34:56Z"},{"title":null,"difficulty":null,"score":500000,"clear_lamp":null,"combo_lamp":null,"full_chain":null,"played_at":"2026-11-26T12:33:56Z"}]}`, rec.Body.String())
}
