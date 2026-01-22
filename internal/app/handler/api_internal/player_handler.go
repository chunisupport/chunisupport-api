package api_internal

import (
	"net/http"

	"github.com/Qman110101/chunisupport-api/internal/app/apierror"
	"github.com/Qman110101/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// PlayerHandler はプレイヤー関連のHTTPリクエストを処理します。
type PlayerHandler struct {
	playerUsecase usecase.PlayerUsecase
}

// NewPlayerHandler は新しいPlayerHandlerを生成します。
func NewPlayerHandler(playerUsecase usecase.PlayerUsecase) *PlayerHandler {
	return &PlayerHandler{playerUsecase: playerUsecase}
}

// createPlayerRequest はプレイヤー作成リクエストのボディの構造です。
type createPlayerRequest struct {
	Name string `json:"name" validate:"required,min=1,max=50"`
}

// CreatePlayer はプレイヤー作成リクエストを処理します。
func (h *PlayerHandler) CreatePlayer(c echo.Context) error {
	req := new(createPlayerRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(req); err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}

	player, err := h.playerUsecase.CreatePlayer(c.Request().Context(), req.Name)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.JSON(http.StatusCreated, player)
}
