package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/service"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// FriendScoreComparisonHandler は承認済みフレンドとの譜面スコア比較を処理します。
type FriendScoreComparisonHandler struct {
	usecase usecase.FriendScoreComparisonUsecase
}

func NewFriendScoreComparisonHandler(u usecase.FriendScoreComparisonUsecase) *FriendScoreComparisonHandler {
	return &FriendScoreComparisonHandler{usecase: u}
}

func (h *FriendScoreComparisonHandler) Get(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}
	username, apiErr := apphandler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	// 大文字正規形だけを許可する。ParseDifficultyPath は小文字変換するため使わない。
	difficulty := c.Param("difficulty")
	if !service.IsExactStandardDifficulty(difficulty) {
		return apierror.ErrInvalidDifficulty
	}

	result, err := h.usecase.Get(c.Request().Context(), user.ID, username, difficulty)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toFriendScoreComparisonResponse(result))
}

func toFriendScoreComparisonResponse(result *usecase.FriendScoreComparisonResult) *internaldto.FriendScoreComparisonResponse {
	if result == nil {
		return nil
	}
	items := make([]internaldto.FriendScoreComparisonItemDTO, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, internaldto.FriendScoreComparisonItemDTO{
			Song: internaldto.FriendScoreComparisonSongDTO{
				ID:     item.Song.ID,
				Title:  item.Song.Title,
				Artist: item.Song.Artist,
			},
			Chart: internaldto.FriendScoreComparisonChartDTO{
				Const:          item.Chart.Const,
				IsConstUnknown: item.Chart.IsConstUnknown,
			},
			Self:            toFriendScoreComparisonRecordDTO(item.Self),
			Friend:          toFriendScoreComparisonRecordDTO(item.Friend),
			ScoreDifference: item.ScoreDifference,
			Result:          item.Result,
		})
	}
	return &internaldto.FriendScoreComparisonResponse{
		Difficulty: result.Difficulty,
		Self: internaldto.FriendScoreComparisonUserDTO{
			Username:   result.Self.Username,
			PlayerName: result.Self.PlayerName,
		},
		Friend: internaldto.FriendScoreComparisonUserDTO{
			Username:   result.Friend.Username,
			PlayerName: result.Friend.PlayerName,
		},
		Summary: internaldto.FriendScoreComparisonSummaryDTO{
			TotalCharts:      result.Summary.TotalCharts,
			SelfWins:         result.Summary.SelfWins,
			Draws:            result.Summary.Draws,
			FriendWins:       result.Summary.FriendWins,
			SelfPlayed:       result.Summary.SelfPlayed,
			FriendPlayed:     result.Summary.FriendPlayed,
			BothPlayed:       result.Summary.BothPlayed,
			SelfOnlyPlayed:   result.Summary.SelfOnlyPlayed,
			FriendOnlyPlayed: result.Summary.FriendOnlyPlayed,
			BothUnplayed:     result.Summary.BothUnplayed,
		},
		Items: items,
	}
}

func toFriendScoreComparisonRecordDTO(record usecase.FriendScoreComparisonRecord) internaldto.FriendScoreComparisonRecordDTO {
	return internaldto.FriendScoreComparisonRecordDTO{
		IsPlayed:  record.IsPlayed,
		Score:     record.Score,
		ClearLamp: record.ClearLamp,
		ComboLamp: record.ComboLamp,
		FullChain: record.FullChain,
		UpdatedAt: record.UpdatedAt,
	}
}
