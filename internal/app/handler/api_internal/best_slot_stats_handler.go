package api_internal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/repository"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// BestSlotStatsHandler はベスト枠採用統計の内部APIを処理します。
type BestSlotStatsHandler struct {
	rankingUsecase usecase.BestSlotRankingUsecase
	statsUsecase   usecase.ChartStatsUsecase
	masterProvider repository.ChartStatsMasterProvider
}

// NewBestSlotStatsHandler はベスト枠採用統計の内部APIハンドラーを生成します。
func NewBestSlotStatsHandler(rankingUsecase usecase.BestSlotRankingUsecase, statsUsecase usecase.ChartStatsUsecase, masterProvider repository.ChartStatsMasterProvider) *BestSlotStatsHandler {
	return &BestSlotStatsHandler{rankingUsecase: rankingUsecase, statsUsecase: statsUsecase, masterProvider: masterProvider}
}

// GetRanking は指定されたベスト枠平均レート帯の譜面採用率ランキングを返します。
func (h *BestSlotStatsHandler) GetRanking(c *echo.Context) error {
	ratingBand := c.QueryParam("rating_band")
	if ratingBand == "" {
		return apierror.ErrValidationFailed.WithInternal(fmt.Errorf("rating_band is required"))
	}
	limit, apiErr := parseBestSlotRankingLimit(c.QueryParam("limit"))
	if apiErr != nil {
		return apiErr
	}
	cursor, apiErr := decodeBestSlotRankingCursor(c.QueryParam("cursor"))
	if apiErr != nil {
		return apiErr
	}
	result, err := h.rankingUsecase.Get(c.Request().Context(), ratingBand, cursor, limit)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	var nextCursor *string
	if result.NextCursor != nil {
		encoded, err := encodeBestSlotRankingCursor(result.NextCursor)
		if err != nil {
			return apierror.ErrInternalError.WithInternal(err)
		}
		nextCursor = &encoded
	}
	return c.JSON(http.StatusOK, internaldto.ToBestSlotRankingResponse(result, nextCursor))
}

// GetChartStats は指定された通常譜面のレート帯別ベスト枠採用統計を返します。
func (h *BestSlotStatsHandler) GetChartStats(c *echo.Context) error {
	displayID, apiErr := apphandler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	difficulty, ok := apphandler.ParseDifficultyPath(c.Param("difficulty"))
	if !ok || difficulty == info.StatsDifficultyWorldsend {
		return apierror.ErrInvalidDifficulty
	}
	requesterAccountTypeID := apphandler.GetRequesterAccountTypeID(c)
	stats, err := h.statsUsecase.GetChartBestSlotStatsByDisplayIDAndDifficulty(c.Request().Context(), displayID, difficulty, requesterAccountTypeID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, internaldto.ToSingleChartBestSlotStatsResponse(stats, h.masterProvider.RatingBands()))
}

func parseBestSlotRankingLimit(value string) (int, *apierror.APIError) {
	if value == "" {
		return info.DefaultBestSlotRankingLimit, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > info.MaxBestSlotRankingLimit {
		return 0, apierror.ErrValidationFailed.WithInternal(fmt.Errorf("limit must be between 1 and %d", info.MaxBestSlotRankingLimit))
	}
	return limit, nil
}

func encodeBestSlotRankingCursor(cursor *repository.BestSlotRankingCursor) (string, error) {
	encoded, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

func decodeBestSlotRankingCursor(value string) (*repository.BestSlotRankingCursor, *apierror.APIError) {
	if value == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, apierror.ErrValidationFailed.WithInternal(fmt.Errorf("invalid cursor: %w", err))
	}
	var cursor repository.BestSlotRankingCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, apierror.ErrValidationFailed.WithInternal(fmt.Errorf("invalid cursor: %w", err))
	}
	difficulty, difficultyOK := info.ParseDifficultyPath(cursor.Difficulty)
	_, displayIDErr := displayid.NewDisplayID(cursor.SongDisplayID)
	if cursor.RatingBand == "" || cursor.BestPlayerPercentage < 0 || cursor.BestPlayerPercentage > 100 || math.IsNaN(cursor.BestPlayerPercentage) || math.IsInf(cursor.BestPlayerPercentage, 0) || cursor.BestPlayerCount < 0 || displayIDErr != nil || !difficultyOK || difficulty == info.StatsDifficultyWorldsend {
		return nil, apierror.ErrValidationFailed.WithInternal(fmt.Errorf("invalid cursor values"))
	}
	cursor.Difficulty = difficulty
	return &cursor, nil
}
