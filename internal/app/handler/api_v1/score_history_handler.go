package api_v1

import (
	"net/http"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	dto "github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// ScoreHistoryHandler は外部API v1のスコア履歴を処理します。
type ScoreHistoryHandler struct {
	usecase usecase.ScoreHistoryUsecase
}

// NewScoreHistoryHandler はスコア履歴Handlerを生成します。
func NewScoreHistoryHandler(scoreHistoryUsecase usecase.ScoreHistoryUsecase) *ScoreHistoryHandler {
	return &ScoreHistoryHandler{usecase: scoreHistoryUsecase}
}

// GetStandard は通常譜面のスコア履歴を返します。
func (h *ScoreHistoryHandler) GetStandard(c echo.Context) error {
	difficulty, ok := handler.ParseDifficultyPath(c.Param("difficulty"))
	if !ok || difficulty == "WORLD'S END" {
		return apierror.ErrInvalidDifficulty
	}
	username, requester, apiErr := scoreHistoryRequestParams(c)
	if apiErr != nil {
		return apiErr
	}
	entries, err := h.usecase.GetStandard(
		c.Request().Context(), username, requester, c.Param("displayid"), difficulty,
	)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toScoreHistoryResponse(entries))
}

// GetWorldsend はWORLD'S END譜面のスコア履歴を返します。
func (h *ScoreHistoryHandler) GetWorldsend(c echo.Context) error {
	username, requester, apiErr := scoreHistoryRequestParams(c)
	if apiErr != nil {
		return apiErr
	}
	entries, err := h.usecase.GetWorldsend(
		c.Request().Context(), username, requester, c.Param("displayid"),
	)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toScoreHistoryResponse(entries))
}

func scoreHistoryRequestParams(c echo.Context) (string, *entity.User, *apierror.APIError) {
	rawUsername := strings.TrimSpace(c.QueryParam("username"))
	if rawUsername == "" {
		return "", nil, apierror.ErrValidationFailedBadRequest
	}
	username, apiErr := handler.ValidateUsername(rawUsername)
	if apiErr != nil {
		return "", nil, apiErr
	}
	requester, _ := c.Get("userEntity").(*entity.User)
	return username, requester, nil
}

func toScoreHistoryResponse(entries []usecase.ScoreHistoryEntry) *dto.ScoreHistoryResponse {
	result := make([]dto.ScoreHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, dto.ScoreHistoryEntry{
			Score: entry.Score, ClearLamp: entry.ClearLamp, ComboLamp: entry.ComboLamp,
			FullChain: entry.FullChain, UpdatedAt: entry.UpdatedAt,
		})
	}
	return &dto.ScoreHistoryResponse{Entries: result}
}
