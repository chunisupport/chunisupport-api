package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type PlayerFavoriteSongHandler struct {
	usecase usecase.PlayerFavoriteSongUsecase
}

func NewPlayerFavoriteSongHandler(u usecase.PlayerFavoriteSongUsecase) *PlayerFavoriteSongHandler {
	return &PlayerFavoriteSongHandler{usecase: u}
}

func (h *PlayerFavoriteSongHandler) List(c *echo.Context) error {
	username, apiErr := apphandler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return apiErr
	}
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	items, err := h.usecase.List(c.Request().Context(), username, requester)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	res := &internaldto.PlayerFavoriteSongsResponse{Items: make([]*internaldto.PlayerFavoriteSongResponseItem, 0, len(items))}
	for _, item := range items {
		res.Items = append(res.Items, &internaldto.PlayerFavoriteSongResponseItem{
			DisplayID:   item.DisplayID,
			Title:       item.Title,
			Jacket:      item.Jacket,
			FavoritedAt: item.FavoritedAt,
		})
	}
	return c.JSON(http.StatusOK, res)
}

func (h *PlayerFavoriteSongHandler) Add(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	var req internaldto.PlayerFavoriteSongRequest
	if err := apphandler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	displayID, err := displayid.NewDisplayID(req.DisplayID)
	if err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	if err := h.usecase.Add(c.Request().Context(), user.ID, displayID); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *PlayerFavoriteSongHandler) Remove(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	displayIDStr := c.Param("displayid")
	displayID, err := displayid.NewDisplayID(displayIDStr)
	if err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	if err := h.usecase.Remove(c.Request().Context(), user.ID, displayID); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
