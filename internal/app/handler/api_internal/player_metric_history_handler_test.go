package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPlayerMetricHistoryUsecase struct {
	get func(context.Context, string, *entity.User) ([]entity.PlayerMetricHistoryEntry, error)
}

func (s *stubPlayerMetricHistoryUsecase) Get(ctx context.Context, username string, requester *entity.User) ([]entity.PlayerMetricHistoryEntry, error) {
	return s.get(ctx, username, requester)
}

func TestPlayerMetricHistoryHandler_Get(t *testing.T) {
	requester := &entity.User{ID: 1}
	collectedAt := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	h := NewPlayerMetricHistoryHandler(&stubPlayerMetricHistoryUsecase{get: func(_ context.Context, username string, gotRequester *entity.User) ([]entity.PlayerMetricHistoryEntry, error) {
		assert.Equal(t, "testuser", username)
		assert.Same(t, requester, gotRequester)
		return []entity.PlayerMetricHistoryEntry{{OfficialRating: 17.25, OfficialOverpower: 12345.67, DataCollectedAt: collectedAt}}, nil
	}})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/internal/users/testuser/rating-op-history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})
	c.Set("userEntity", requester)

	err := h.Get(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"entries":[{"rating":17.25,"overpower":12345.67,"data_collected_at":"2026-08-08T12:00:00Z"}]}`, rec.Body.String())
}
