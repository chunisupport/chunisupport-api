package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/domain/vo/displayid"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

const (
	// MaxPlayerLockedSongBatchItems is the maximum number of items allowed in a batch operation
	MaxPlayerLockedSongBatchItems = 1000
)

type PlayerLockedSongHandler struct {
	usecase usecase.PlayerLockedSongUsecase
}

func NewPlayerLockedSongHandler(u usecase.PlayerLockedSongUsecase) *PlayerLockedSongHandler {
	return &PlayerLockedSongHandler{usecase: u}
}

func (h *PlayerLockedSongHandler) List(c echo.Context) error {
	username := c.Param("username")
	var requester *entity.User
	if userEntity, ok := c.Get("userEntity").(*entity.User); ok {
		requester = userEntity
	}

	items, err := h.usecase.List(c.Request().Context(), username, requester)
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
	displayID, err := displayid.NewDisplayID(req.DisplayID)
	if err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	if err := h.usecase.Lock(c.Request().Context(), user.ID, &usecase.PlayerLockedSongInput{DisplayID: displayID, IsUltima: req.IsUltima}); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *PlayerLockedSongHandler) Unlock(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	var req internaldto.PlayerLockedSongUnlockRequest
	if err := c.Bind(&req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	displayID, err := displayid.NewDisplayID(req.DisplayID)
	if err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	if err := h.usecase.Unlock(c.Request().Context(), user.ID, &usecase.PlayerLockedSongInput{DisplayID: displayID, IsUltima: bool(req.IsUltima)}); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *PlayerLockedSongHandler) Batch(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	var req internaldto.PlayerLockedSongBatchRequest
	if err := apphandler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	// Validate total batch size before processing
	totalItems := len(req.Add) + len(req.Delete)
	if totalItems > MaxPlayerLockedSongBatchItems {
		return apierror.ErrValidationFailed
	}
	toInputs := func(items []*internaldto.PlayerLockedSongBatchRequestItem) ([]*usecase.PlayerLockedSongInput, error) {
		inputs := make([]*usecase.PlayerLockedSongInput, 0, len(items))
		for _, item := range items {
			if item == nil {
				return nil, apierror.ErrValidationFailed
			}
			displayID, err := displayid.NewDisplayID(item.DisplayID)
			if err != nil {
				return nil, apierror.ErrValidationFailed.WithInternal(err)
			}
			inputs = append(inputs, &usecase.PlayerLockedSongInput{DisplayID: displayID, IsUltima: item.IsUltima})
		}
		return inputs, nil
	}
	addInputs, err := toInputs(req.Add)
	if err != nil {
		return err
	}
	deleteInputs, err := toInputs(req.Delete)
	if err != nil {
		return err
	}
	if err := h.usecase.Batch(c.Request().Context(), user.ID, &usecase.PlayerLockedSongBatchInput{Add: addInputs, Delete: deleteInputs}); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
