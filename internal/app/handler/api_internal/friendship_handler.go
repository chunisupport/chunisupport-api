package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type FriendshipHandler struct {
	usecase usecase.FriendshipUsecase
}

func NewFriendshipHandler(u usecase.FriendshipUsecase) *FriendshipHandler {
	return &FriendshipHandler{usecase: u}
}

func (h *FriendshipHandler) SendRequest(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}
	var req internaldto.FriendRequestCreateRequest
	if err := apphandler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	if err := h.usecase.SendRequest(c.Request().Context(), user.ID, req.Username); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *FriendshipHandler) ListFriends(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}
	items, err := h.usecase.ListFriends(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toFriendshipListResponse(items))
}

func (h *FriendshipHandler) ListReceivedRequests(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}
	items, err := h.usecase.ListReceivedRequests(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toFriendshipListResponse(items))
}

func (h *FriendshipHandler) ListSentRequests(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}
	items, err := h.usecase.ListSentRequests(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toFriendshipListResponse(items))
}

func (h *FriendshipHandler) AcceptRequest(c *echo.Context) error {
	user, requesterUsername, err := h.authenticatedUserAndPathUsername(c)
	if err != nil {
		return err
	}
	if err := h.usecase.AcceptRequest(c.Request().Context(), user.ID, requesterUsername); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *FriendshipHandler) RejectRequest(c *echo.Context) error {
	user, requesterUsername, err := h.authenticatedUserAndPathUsername(c)
	if err != nil {
		return err
	}
	if err := h.usecase.RejectRequest(c.Request().Context(), user.ID, requesterUsername); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *FriendshipHandler) CancelRequest(c *echo.Context) error {
	user, targetUsername, err := h.authenticatedUserAndPathUsername(c)
	if err != nil {
		return err
	}
	if err := h.usecase.CancelRequest(c.Request().Context(), user.ID, targetUsername); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *FriendshipHandler) Remove(c *echo.Context) error {
	user, friendUsername, err := h.authenticatedUserAndPathUsername(c)
	if err != nil {
		return err
	}
	if err := h.usecase.Remove(c.Request().Context(), user.ID, friendUsername); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *FriendshipHandler) authenticatedUserAndPathUsername(c *echo.Context) (*entity.User, string, error) {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return nil, "", err
	}
	username, apiErr := apphandler.ValidateUsername(c.Param("username"))
	if apiErr != nil {
		return nil, "", apiErr
	}
	return user, username, nil
}

func toFriendshipListResponse(items []*usecase.FriendshipUserOutput) *internaldto.FriendshipListResponse {
	res := &internaldto.FriendshipListResponse{
		Items: make([]*internaldto.FriendshipUserResponse, 0, len(items)),
	}
	for _, item := range items {
		res.Items = append(res.Items, &internaldto.FriendshipUserResponse{
			Username:    item.Username,
			PlayerLevel: item.PlayerLevel,
			PlayerName:  item.PlayerName,
			Rating:      item.Rating,
			IsPrivate:   item.IsPrivate,
			RequestedAt: item.RequestedAt,
			AcceptedAt:  item.AcceptedAt,
		})
	}
	return res
}
