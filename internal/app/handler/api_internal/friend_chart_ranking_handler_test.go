package api_internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

type stubFriendChartRankingUsecase struct{}

func (stubFriendChartRankingUsecase) GetStandard(context.Context, int, string, string) (*usecase.FriendChartRankingResult, error) {
	return nil, nil
}

func TestFriendChartRankingHandler_不正なDisplayIDはHTTP422を返す(t *testing.T) {
	// Given
	e := echo.New()
	e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
	handler := NewFriendChartRankingHandler(stubFriendChartRankingUsecase{})
	e.GET("/internal/friend-rankings/songs/:displayid/charts/:difficulty", func(c *echo.Context) error {
		c.Set("userEntity", &entity.User{ID: 1})
		return handler.GetStandard(c)
	})
	req := httptest.NewRequest(http.MethodGet, "/internal/friend-rankings/songs/invalid/charts/MASTER", nil)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.JSONEq(t, `{"error":{"status":422,"code":"validation_failed","message":"入力値の形式を確認してください。"}}`, rec.Body.String())
}

func (stubFriendChartRankingUsecase) GetWorldsend(context.Context, int, string) (*usecase.FriendChartRankingResult, error) {
	return nil, nil
}
