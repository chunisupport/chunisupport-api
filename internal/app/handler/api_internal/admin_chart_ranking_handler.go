package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type AdminChartRankingHandler struct {
	usecase usecase.AdminChartRankingUsecase
}

func NewAdminChartRankingHandler(u usecase.AdminChartRankingUsecase) *AdminChartRankingHandler {
	return &AdminChartRankingHandler{usecase: u}
}

func (h *AdminChartRankingHandler) GetStandard(c *echo.Context) error {
	displayID, apiErr := apphandler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	difficulty, ok := apphandler.ParseDifficultyPath(c.Param("difficulty"))
	if !ok || difficulty == "WORLD'S END" {
		return apierror.ErrInvalidDifficulty
	}

	result, err := h.usecase.GetStandard(c.Request().Context(), displayID, difficulty)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toAdminChartRankingResponse(result))
}

func (h *AdminChartRankingHandler) GetWorldsend(c *echo.Context) error {
	displayID, apiErr := apphandler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}

	result, err := h.usecase.GetWorldsend(c.Request().Context(), displayID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toAdminChartRankingResponse(result))
}

func toAdminChartRankingResponse(result *usecase.AdminChartRankingResult) *internaldto.AdminChartRankingResponse {
	if result == nil {
		return nil
	}

	response := &internaldto.AdminChartRankingResponse{
		Song: internaldto.AdminChartRankingSongDTO{
			ID:     result.Song.ID,
			Title:  result.Song.Title,
			Artist: result.Song.Artist,
		},
		Chart: internaldto.AdminChartRankingChartDTO{
			Difficulty:  result.Chart.Difficulty,
			LevelStar:   result.Chart.LevelStar,
			Attribute:   result.Chart.Attribute,
			IsWorldsend: result.Chart.IsWorldsend,
		},
		Ranking: make([]internaldto.AdminChartRankingEntryDTO, 0, len(result.Ranking)),
		Total:   result.Total,
	}
	if !result.Chart.IsWorldsend {
		response.Chart.Const = &result.Chart.Const
		response.Chart.IsConstUnknown = &result.Chart.IsConstUnknown
	}

	for _, entry := range result.Ranking {
		item := internaldto.AdminChartRankingEntryDTO{
			Rank:       entry.Rank,
			Username:   entry.Username,
			PlayerName: entry.PlayerName,
			Score:      entry.Score,
			ClearLamp:  entry.ClearLamp,
			ComboLamp:  entry.ComboLamp,
			FullChain:  entry.FullChain,
			UpdatedAt:  entry.UpdatedAt,
		}
		if !result.Chart.IsWorldsend {
			rating := entry.Rating
			overpower := entry.Overpower
			overpowerPercent := entry.OverpowerPercent
			item.Rating = &rating
			item.Overpower = &overpower
			item.OverpowerPercent = &overpowerPercent
		}
		response.Ranking = append(response.Ranking, item)
	}

	return response
}
