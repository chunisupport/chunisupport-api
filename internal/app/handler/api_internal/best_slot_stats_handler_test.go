package api_internal

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/chartconstant"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/ratingband"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bestSlotRankingUsecaseStub struct{}

func (bestSlotRankingUsecaseStub) Get(_ context.Context, ratingBand string, _ *repository.BestSlotRankingCursor, limit int) (*usecase.BestSlotRankingResult, error) {
	return &usecase.BestSlotRankingResult{
		RatingBand: ratingBand,
		Ranking: []usecase.BestSlotRankingEntry{{
			Rank: 1, SongID: "0000000000000001", Title: "楽曲名", Difficulty: "MASTER",
			ChartConst: chartconstant.ChartConstant(14.8), BestPlayerCount: 10, BestPlayerPercentage: 25,
		}},
	}, nil
}

type chartStatsUsecaseBestSlotStub struct{}

type bestSlotMasterProviderStub struct{}

func (bestSlotMasterProviderStub) RatingBands() []*ratingband.RatingBand { return nil }

func (chartStatsUsecaseBestSlotStub) GetSongStatsByDisplayID(context.Context, string, *int) (*entity.SongChartStats, error) {
	return nil, nil
}
func (chartStatsUsecaseBestSlotStub) GetChartStatsByDisplayIDAndDifficulty(context.Context, string, string, *int) (*entity.SingleChartStats, error) {
	return nil, nil
}
func (chartStatsUsecaseBestSlotStub) GetChartBestSlotStatsByDisplayIDAndDifficulty(context.Context, string, string, *int) (*entity.SingleChartBestSlotStats, error) {
	return nil, nil
}

func TestBestSlotStatsHandler_GetRanking_不要な楽曲情報と内部IDを返さない(t *testing.T) {
	// Given
	e := echo.New()
	h := NewBestSlotStatsHandler(bestSlotRankingUsecaseStub{}, chartStatsUsecaseBestSlotStub{}, bestSlotMasterProviderStub{})
	e.GET("/internal/best-slot-rankings", h.GetRanking)
	req := httptest.NewRequest(http.MethodGet, "/internal/best-slot-rankings?rating_band=17.0", nil)
	rec := httptest.NewRecorder()

	// When
	e.ServeHTTP(rec, req)

	// Then
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"rating_band":"17.0"`)
	assert.Contains(t, rec.Body.String(), `"title":"楽曲名"`)
	assert.NotContains(t, rec.Body.String(), "rating_band_id")
	assert.NotContains(t, rec.Body.String(), "artist")
	assert.NotContains(t, rec.Body.String(), "jacket")
}

func TestBestSlotRankingCursor_公開値だけで往復する(t *testing.T) {
	// Given
	cursor := &repository.BestSlotRankingCursor{
		RatingBand: "17.0", BestPlayerPercentage: 25, BestPlayerCount: 10,
		SongDisplayID: "0000000000000001", Difficulty: "MASTER",
	}

	// When
	encoded, err := encodeBestSlotRankingCursor(cursor)
	require.NoError(t, err)
	decoded, apiErr := decodeBestSlotRankingCursor(encoded)

	// Then
	require.Nil(t, apiErr)
	assert.Equal(t, cursor, decoded)
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "ChartID")
	assert.NotContains(t, string(payload), "RatingBandID")
}

func TestDecodeBestSlotRankingCursor_破損した値はエラー(t *testing.T) {
	// When
	decoded, apiErr := decodeBestSlotRankingCursor("not-base64!")

	// Then
	assert.Nil(t, decoded)
	require.NotNil(t, apiErr)
}

func TestDecodeBestSlotRankingCursor_難易度を大文字へ正規化する(t *testing.T) {
	// Given
	cursor := &repository.BestSlotRankingCursor{
		RatingBand: "17.0", BestPlayerPercentage: 25, BestPlayerCount: 10,
		SongDisplayID: "0000000000000001", Difficulty: "master",
	}
	encoded, err := encodeBestSlotRankingCursor(cursor)
	require.NoError(t, err)

	// When
	decoded, apiErr := decodeBestSlotRankingCursor(encoded)

	// Then
	require.Nil(t, apiErr)
	require.NotNil(t, decoded)
	assert.Equal(t, "MASTER", decoded.Difficulty)
}
