package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubInternalScoreHistoryUsecase struct {
	getStandard func(context.Context, string, *entity.User, string, string) ([]usecase.ScoreHistoryEntry, error)
}

func (s *stubInternalScoreHistoryUsecase) GetStandard(ctx context.Context, username string, requester *entity.User, displayID, difficulty string) ([]usecase.ScoreHistoryEntry, error) {
	return s.getStandard(ctx, username, requester, displayID, difficulty)
}

func (s *stubInternalScoreHistoryUsecase) GetWorldsend(context.Context, string, *entity.User, string) ([]usecase.ScoreHistoryEntry, error) {
	return nil, nil
}

func TestInternalScoreHistoryHandler_GetStandard(t *testing.T) {
	t.Run("Firebase認証ユーザーを閲覧者として渡す", func(t *testing.T) {
		requester := &entity.User{ID: 1}
		h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{
			getStandard: func(_ context.Context, username string, gotRequester *entity.User, displayID, difficulty string) ([]usecase.ScoreHistoryEntry, error) {
				assert.Equal(t, "testuser", username)
				assert.Same(t, requester, gotRequester)
				assert.Equal(t, "song", displayID)
				assert.Equal(t, "MASTER", difficulty)
				return []usecase.ScoreHistoryEntry{}, nil
			},
		})
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/internal/songs/song/score-history/master?username=testuser", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/internal/songs/:displayid/score-history/:difficulty")
		c.SetParamNames("displayid", "difficulty")
		c.SetParamValues("song", "master")
		c.Set("userEntity", requester)

		err := h.GetStandard(c)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("ユーザー名なしは400を返す", func(t *testing.T) {
		h := NewScoreHistoryHandler(&stubInternalScoreHistoryUsecase{})
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/internal/songs/song/score-history/master", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("displayid", "difficulty")
		c.SetParamValues("song", "master")

		err := h.GetStandard(c)

		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
	})
}
