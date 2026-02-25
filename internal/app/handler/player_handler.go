package handler

import (
	"errors"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
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
	Name string `json:"name" validate:"required,min=1,max=20,excludesall=abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"`
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

	user, ok := c.Get("userEntity").(*entity.User)
	if !ok || user == nil {
		return apierror.ErrUnauthorized.WithInternal(errors.New("user entity not found in context"))
	}

	player, err := h.playerUsecase.CreatePlayer(c.Request().Context(), user.ID, req.Name)
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.JSON(http.StatusCreated, player)
}
