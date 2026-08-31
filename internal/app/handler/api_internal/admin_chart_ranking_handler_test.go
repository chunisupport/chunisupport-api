package api_internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appmiddleware "github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAdminChartRankingUsecase struct {
	standardResult  *usecase.AdminChartRankingResult
	worldsendResult *usecase.AdminChartRankingResult
}

func (s stubAdminChartRankingUsecase) GetStandard(context.Context, string, string) (*usecase.AdminChartRankingResult, error) {
	return s.standardResult, nil
}

func (s stubAdminChartRankingUsecase) GetWorldsend(context.Context, string) (*usecase.AdminChartRankingResult, error) {
	return s.worldsendResult, nil
}

func TestAdminChartRankingHandler_不正なパスパラメータはHTTP422を返す(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantCode   string
	}{
		{name: "display IDが不正", path: "/internal/admin/chart-rankings/songs/invalid/charts/MASTER", wantStatus: http.StatusUnprocessableEntity, wantCode: "validation_failed"},
		{name: "難易度が不正", path: "/internal/admin/chart-rankings/songs/0000000000000001/charts/WORLD'S%20END", wantStatus: http.StatusBadRequest, wantCode: "invalid_difficulty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			e.HTTPErrorHandler = appmiddleware.CustomHTTPErrorHandler
			handler := NewAdminChartRankingHandler(stubAdminChartRankingUsecase{})
			e.GET("/internal/admin/chart-rankings/songs/:displayid/charts/:difficulty", handler.GetStandard)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			// When
			e.ServeHTTP(rec, req)

			// Then
			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Contains(t, rec.Body.String(), `"code":"`+tt.wantCode+`"`)
		})
	}
}

func TestAdminChartRankingResponse_管理者向け項目だけを返す(t *testing.T) {
	// Given
	result := &usecase.AdminChartRankingResult{
		Chart: usecase.AdminChartRankingChart{Difficulty: "MASTER"},
		Ranking: []usecase.AdminChartRankingEntry{{
			Rank:       1,
			Username:   "ranking-user",
			PlayerName: "PLAYER",
			Score:      1_010_000,
		}},
		Total: 123,
	}

	// When
	body, err := json.Marshal(toAdminChartRankingResponse(result))

	// Then
	require.NoError(t, err)
	assert.Contains(t, string(body), `"username":"ranking-user"`)
	assert.Contains(t, string(body), `"total":123`)
	assert.NotContains(t, string(body), "user_id")
	assert.NotContains(t, string(body), "is_self")
	assert.NotContains(t, string(body), "my_rank")
}

func TestAdminChartRankingResponse_WORLD送信ではレート項目を返さない(t *testing.T) {
	// Given
	result := &usecase.AdminChartRankingResult{
		Chart:   usecase.AdminChartRankingChart{Difficulty: "WORLD'S END", IsWorldsend: true},
		Ranking: []usecase.AdminChartRankingEntry{{Rank: 1, Username: "ranking-user", Score: 1_010_000}},
		Total:   1,
	}

	// When
	body, err := json.Marshal(toAdminChartRankingResponse(result))

	// Then
	require.NoError(t, err)
	assert.NotContains(t, string(body), `"const"`)
	assert.NotContains(t, string(body), `"rating"`)
	assert.NotContains(t, string(body), `"overpower"`)
}
