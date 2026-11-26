package api_internal

import (
	"net/http"

	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// PlayerPlayHistoryHandler は管理画面向けプレイ履歴APIを処理します。
type PlayerPlayHistoryHandler struct {
	usecase usecase.PlayerPlayHistoryUsecase
}

// NewPlayerPlayHistoryHandler はプレイ履歴APIハンドラーを生成します。
func NewPlayerPlayHistoryHandler(playHistoryUsecase usecase.PlayerPlayHistoryUsecase) *PlayerPlayHistoryHandler {
	return &PlayerPlayHistoryHandler{usecase: playHistoryUsecase}
}

// Get は指定ユーザーの直近プレイ履歴を返します。
func (h *PlayerPlayHistoryHandler) Get(c *echo.Context) error {
	username, apiErr := apphandler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}

	entries, err := h.usecase.Get(c.Request().Context(), username, apphandler.Requester(c))
	if err != nil {
		return handleUserProfileError(c.Request().Context(), err, username, "user play history")
	}

	c.Response().Header().Set(echo.HeaderCacheControl, "private, no-store")
	response := &internaldto.PlayerPlayHistoryResponse{Entries: make([]internaldto.PlayerPlayHistoryEntry, 0, len(entries))}
	for _, entry := range entries {
		response.Entries = append(response.Entries, internaldto.PlayerPlayHistoryEntry{
			Title: entry.Title, Difficulty: entry.Difficulty, Score: entry.Score,
			ClearLamp: entry.ClearLamp, ComboLamp: entry.ComboLamp, FullChain: entry.FullChain,
			PlayedAt: entry.PlayedAt,
		})
	}
	return c.JSON(http.StatusOK, response)
}
