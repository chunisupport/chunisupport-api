package api_internal

import (
	"net/http"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// ScoreHistoryHandler はinternal APIのスコア履歴取得を処理します。
type ScoreHistoryHandler struct {
	usecase usecase.ScoreHistoryUsecase
}

// NewScoreHistoryHandler はinternal API用のスコア履歴Handlerを生成します。
func NewScoreHistoryHandler(scoreHistoryUsecase usecase.ScoreHistoryUsecase) *ScoreHistoryHandler {
	return &ScoreHistoryHandler{usecase: scoreHistoryUsecase}
}

// GetStandard は通常譜面のスコア履歴を返します。
func (h *ScoreHistoryHandler) GetStandard(c echo.Context) error {
	difficulty, ok := apphandler.ParseDifficultyPath(c.Param("difficulty"))
	if !ok || difficulty == "WORLD'S END" {
		return apierror.ErrInvalidDifficulty
	}
	username, requester, apiErr := internalScoreHistoryRequestParams(c)
	if apiErr != nil {
		return apiErr
	}
	entries, err := h.usecase.GetStandard(
		c.Request().Context(), username, requester, c.Param("displayid"), difficulty,
	)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toInternalScoreHistoryResponse(entries))
}

// GetWorldsend はWORLD'S END譜面のスコア履歴を返します。
func (h *ScoreHistoryHandler) GetWorldsend(c echo.Context) error {
	username, requester, apiErr := internalScoreHistoryRequestParams(c)
	if apiErr != nil {
		return apiErr
	}
	entries, err := h.usecase.GetWorldsend(
		c.Request().Context(), username, requester, c.Param("displayid"),
	)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toInternalScoreHistoryResponse(entries))
}

func internalScoreHistoryRequestParams(c echo.Context) (string, *entity.User, *apierror.APIError) {
	rawUsername := strings.TrimSpace(c.QueryParam("username"))
	if rawUsername == "" {
		return "", nil, apierror.ErrValidationFailedBadRequest
	}
	username, apiErr := apphandler.ValidateUsername(rawUsername)
	if apiErr != nil {
		return "", nil, apiErr
	}
	requester, _ := c.Get("userEntity").(*entity.User)
	return username, requester, nil
}

func toInternalScoreHistoryResponse(entries []usecase.ScoreHistoryEntry) *internaldto.ScoreHistoryResponse {
	result := make([]internaldto.ScoreHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, internaldto.ScoreHistoryEntry{
			Score: entry.Score, ClearLamp: entry.ClearLamp, ComboLamp: entry.ComboLamp,
			FullChain: entry.FullChain, UpdatedAt: entry.UpdatedAt,
		})
	}
	return &internaldto.ScoreHistoryResponse{Entries: result}
}
