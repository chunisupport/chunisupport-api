package api_v1

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

type stubScoreHistoryUsecase struct {
	getStandard func(context.Context, string, *entity.User, string, string) ([]usecase.ScoreHistoryEntry, error)
}

func (s *stubScoreHistoryUsecase) GetStandard(ctx context.Context, username string, requester *entity.User, displayID, difficulty string) ([]usecase.ScoreHistoryEntry, error) {
	return s.getStandard(ctx, username, requester, displayID, difficulty)
}

func (s *stubScoreHistoryUsecase) GetWorldsend(context.Context, string, *entity.User, string) ([]usecase.ScoreHistoryEntry, error) {
	return nil, nil
}

func TestScoreHistoryHandler_GetStandard(t *testing.T) {
	t.Run("username未指定は400のvalidation_failed", func(t *testing.T) {
		h := NewScoreHistoryHandler(&stubScoreHistoryUsecase{})
		c := newScoreHistoryContext("/v1/songs/song/score-history/master", "song", "master", "")

		err := h.GetStandard(c)

		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeValidationFailed, apiErr.Code)
	})

	t.Run("非対応難易度は400の専用コード", func(t *testing.T) {
		h := NewScoreHistoryHandler(&stubScoreHistoryUsecase{
			getStandard: func(context.Context, string, *entity.User, string, string) ([]usecase.ScoreHistoryEntry, error) {
				return nil, usecase.ErrScoreHistoryUnsupportedDifficulty
			},
		})
		c := newScoreHistoryContext("/v1/songs/song/score-history/basic?username=testuser", "song", "basic", "testuser")

		err := h.GetStandard(c)

		apiErr, ok := err.(*apierror.APIError)
		require.True(t, ok)
		assert.Equal(t, http.StatusBadRequest, apiErr.HTTPStatus)
		assert.Equal(t, apierror.CodeScoreHistoryUnsupportedDifficulty, apiErr.Code)
	})
}

func newScoreHistoryContext(target, displayID, difficulty, username string) echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("displayid", "difficulty")
	c.SetParamValues(displayID, difficulty)
	if username != "" {
		query := req.URL.Query()
		query.Set("username", username)
		req.URL.RawQuery = query.Encode()
	}
	return c
}
