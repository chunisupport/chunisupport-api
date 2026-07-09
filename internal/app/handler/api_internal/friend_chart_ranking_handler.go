package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// FriendChartRankingHandler は譜面単位フレンドランキング取得を処理します。
type FriendChartRankingHandler struct {
	usecase usecase.FriendChartRankingUsecase
}

func NewFriendChartRankingHandler(u usecase.FriendChartRankingUsecase) *FriendChartRankingHandler {
	return &FriendChartRankingHandler{usecase: u}
}

func (h *FriendChartRankingHandler) GetStandard(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}
	displayID, apiErr := apphandler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	difficulty, ok := apphandler.ParseDifficultyPath(c.Param("difficulty"))
	if !ok || difficulty == "WORLD'S END" {
		return apierror.ErrInvalidDifficulty
	}

	result, err := h.usecase.GetStandard(c.Request().Context(), user.ID, displayID, difficulty)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toFriendChartRankingResponse(result))
}

func toFriendChartRankingResponse(result *usecase.FriendChartRankingResult) *internaldto.FriendChartRankingResponse {
	if result == nil {
		return nil
	}
	res := &internaldto.FriendChartRankingResponse{
		Song: internaldto.FriendChartRankingSongDTO{
			ID:     result.Song.ID,
			Title:  result.Song.Title,
			Artist: result.Song.Artist,
		},
		Chart: internaldto.FriendChartRankingChartDTO{
			Difficulty:     result.Chart.Difficulty,
			Const:          result.Chart.Const,
			IsConstUnknown: result.Chart.IsConstUnknown,
		},
		Ranking: make([]internaldto.FriendChartRankingEntryDTO, 0, len(result.Ranking)),
		MyRank:  result.MyRank,
		Total:   result.Total,
	}
	for _, entry := range result.Ranking {
		res.Ranking = append(res.Ranking, internaldto.FriendChartRankingEntryDTO{
			Rank:             entry.Rank,
			UserID:           entry.UserID,
			Username:         entry.Username,
			PlayerName:       entry.PlayerName,
			Score:            entry.Score,
			Rating:           entry.Rating,
			Overpower:        entry.Overpower,
			OverpowerPercent: entry.OverpowerPercent,
			ClearLamp:        entry.ClearLamp,
			ComboLamp:        entry.ComboLamp,
			FullChain:        entry.FullChain,
			UpdatedAt:        entry.UpdatedAt,
			IsSelf:           entry.IsSelf,
		})
	}
	return res
}
