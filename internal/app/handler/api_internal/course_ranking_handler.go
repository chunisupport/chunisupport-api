package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// CourseRankingHandler は閲覧者の権限に応じたコースランキングを返します。
type CourseRankingHandler struct {
	usecase usecase.CourseRankingUsecase
}

// NewCourseRankingHandler はフレンド用と管理者用のルートで共有するハンドラーを作成します。
func NewCourseRankingHandler(u usecase.CourseRankingUsecase) *CourseRankingHandler {
	return &CourseRankingHandler{usecase: u}
}

func (h *CourseRankingHandler) GetFriends(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}
	displayID, apiErr := apphandler.ValidateDisplayID(c.Param("id"))
	if apiErr != nil {
		return apiErr
	}
	result, err := h.usecase.GetFriends(c.Request().Context(), user.ID, displayID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return writeCourseRanking(c, result, true)
}

func (h *CourseRankingHandler) GetAdmin(c *echo.Context) error {
	displayID, apiErr := apphandler.ValidateDisplayID(c.Param("id"))
	if apiErr != nil {
		return apiErr
	}
	result, err := h.usecase.GetAdmin(c.Request().Context(), displayID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return writeCourseRanking(c, result, false)
}

func writeCourseRanking(c *echo.Context, result *usecase.CourseRankingResult, includeIsSelf bool) error {
	response := internaldto.CourseRankingResponse{
		Course:  internaldto.CourseRankingCourseDTO{ID: result.CourseID, Name: result.Name, Class: result.Class},
		Ranking: make([]internaldto.CourseRankingEntryDTO, 0, len(result.Ranking)),
		Total:   result.Total,
	}
	for _, entry := range result.Ranking {
		item := internaldto.CourseRankingEntryDTO{
			CourseScoreHistoryEntry: internaldto.CourseScoreHistoryEntry{
				Score: int(entry.Score), IsClear: entry.IsClear, ComboLamp: entry.ComboLamp, UpdatedAt: entry.UpdatedAt,
			},
			Rank: entry.Rank, Username: entry.Username, PlayerName: entry.PlayerName,
		}
		if includeIsSelf {
			item.IsSelf = new(entry.IsSelf)
		}
		response.Ranking = append(response.Ranking, item)
	}
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.JSON(http.StatusOK, response)
}
