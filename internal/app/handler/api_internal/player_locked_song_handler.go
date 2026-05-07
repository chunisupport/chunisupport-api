package api_internal

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

type PlayerLockedSongHandler struct {
	usecase usecase.PlayerLockedSongUsecase
}

func NewPlayerLockedSongHandler(u usecase.PlayerLockedSongUsecase) *PlayerLockedSongHandler {
	return &PlayerLockedSongHandler{usecase: u}
}

func (h *PlayerLockedSongHandler) List(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	items, err := h.usecase.List(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	res := &internaldto.PlayerLockedSongsResponse{Items: make([]*internaldto.PlayerLockedSongResponseItem, 0, len(items))}
	for _, item := range items {
		res.Items = append(res.Items, &internaldto.PlayerLockedSongResponseItem{DisplayID: item.DisplayID, Title: item.Title, IsUltima: item.IsUltima})
	}
	return c.JSON(http.StatusOK, res)
}

func (h *PlayerLockedSongHandler) Lock(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	var req internaldto.PlayerLockedSongRequest
	if err := apphandler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	if err := h.usecase.Lock(c.Request().Context(), user.ID, &usecase.PlayerLockedSongInput{DisplayID: req.DisplayID, IsUltima: req.IsUltima}); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *PlayerLockedSongHandler) Unlock(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	isUltima := false
	if raw, ok := c.QueryParams()["is_ultima"]; ok {
		if len(raw) == 0 || raw[0] == "" {
			return apierror.ErrBadRequest
		}
		v, err := strconv.ParseBool(raw[0])
		if err != nil {
			return apierror.ErrBadRequest.WithInternal(err)
		}
		isUltima = v
	}
	displayID := c.Param("displayid")
	if len(displayID) != 16 {
		return apierror.ErrValidationFailed
	}
	if err := h.usecase.Unlock(c.Request().Context(), user.ID, &usecase.PlayerLockedSongInput{DisplayID: displayID, IsUltima: isUltima}); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
