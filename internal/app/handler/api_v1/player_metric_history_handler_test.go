package api_v1

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
	collectedAt := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	oldCollectedAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	overpowerPercent := 98.76
	h := NewPlayerMetricHistoryHandler(&stubPlayerMetricHistoryUsecase{get: func(_ context.Context, username string, requester *entity.User) ([]entity.PlayerMetricHistoryEntry, error) {
		assert.Equal(t, "testuser", username)
		assert.Nil(t, requester)
		return []entity.PlayerMetricHistoryEntry{
			{OfficialRating: 17.25, OfficialOverpower: 12345.67, OfficialOverpowerPercent: &overpowerPercent, DataCollectedAt: collectedAt},
			{OfficialRating: 17.20, OfficialOverpower: 12300.00, OfficialOverpowerPercent: nil, DataCollectedAt: oldCollectedAt},
		}, nil
	}})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/users/testuser/rating-op-history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPathValues(echo.PathValues{{Name: "username", Value: "testuser"}})

	err := h.Get(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"entries":[{"rating":17.25,"overpower":12345.67,"overpower_percent":98.76,"data_collected_at":"2026-08-08T12:00:00Z"},{"rating":17.2,"overpower":12300,"overpower_percent":null,"data_collected_at":"2026-07-01T12:00:00Z"}]}`, rec.Body.String())
}
