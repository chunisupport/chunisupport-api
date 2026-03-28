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
	dtoapiinternal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockOverpowerSummaryUsecase struct {
	mock.Mock
}

func (m *mockOverpowerSummaryUsecase) Get(ctx context.Context, user *entity.User) (*dtoapiinternal.OverpowerSummaryResponse, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dtoapiinternal.OverpowerSummaryResponse), args.Error(1)
}

func TestOverpowerSummaryHandlerGet(t *testing.T) {
	e := newTestEcho()
	mockUsecase := new(mockOverpowerSummaryUsecase)
	h := api_internal.NewOverpowerSummaryHandler(mockUsecase)
	user := &entity.User{ID: 1}
	updatedAt := time.Date(2026, 3, 25, 12, 34, 56, 0, time.UTC)

	t.Run("認証済みなら200で集計結果を返す", func(t *testing.T) {
		expected := &dtoapiinternal.OverpowerSummaryResponse{
			UpdatedAt: updatedAt,
			Overall: dtoapiinternal.OverpowerSummaryItem{
				CurrentOP:   123.45,
				MaxOP:       200,
				Percent:     61.725,
				TargetCount: 10,
				PlayedCount: 8,
			},
			Genres: map[string]dtoapiinternal.OverpowerSummaryItem{
				"POPS & ANIME": {CurrentOP: 12.3, MaxOP: 20},
			},
			Difficulties: map[string]dtoapiinternal.OverpowerSummaryItem{
				"MASTER": {CurrentOP: 45.6, MaxOP: 60},
			},
			Levels: map[string]dtoapiinternal.OverpowerSummaryItem{
				"14+": {CurrentOP: 30.1, MaxOP: 40},
			},
		}
		mockUsecase.On("Get", mock.Anything, user).Return(expected, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/internal/me/overpower-summary", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		err := h.Get(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var body dtoapiinternal.OverpowerSummaryResponse
		assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.True(t, expected.UpdatedAt.Equal(body.UpdatedAt))
		assert.InDelta(t, expected.Overall.CurrentOP, body.Overall.CurrentOP, 0.0001)
		mockUsecase.AssertExpectations(t)
	})

	t.Run("未認証なら401を返す", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/internal/me/overpower-summary", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.Get(c)

		assert.ErrorIs(t, err, apierror.ErrUnauthorized)
	})

	t.Run("プレイヤー未連携は404へ変換する", func(t *testing.T) {
		mockUsecase.On("Get", mock.Anything, user).Return((*dtoapiinternal.OverpowerSummaryResponse)(nil), usecase.ErrPlayerNotLinked).Once()

		req := httptest.NewRequest(http.MethodGet, "/internal/me/overpower-summary", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("userEntity", user)

		err := h.Get(c)

		var apiErr *apierror.APIError
		if assert.ErrorAs(t, err, &apiErr) {
			assert.Equal(t, apierror.CodeUserNotFound, apiErr.Code)
		}
		mockUsecase.AssertExpectations(t)
	})
}
