package api_internal

import (
	"net/http"
	"strconv"

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
	user, requesterID, err := h.authenticatedUserAndPathUserID(c)
	if err != nil {
		return err
	}
	if err := h.usecase.AcceptRequest(c.Request().Context(), user.ID, requesterID); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *FriendshipHandler) RejectRequest(c *echo.Context) error {
	user, requesterID, err := h.authenticatedUserAndPathUserID(c)
	if err != nil {
		return err
	}
	if err := h.usecase.RejectRequest(c.Request().Context(), user.ID, requesterID); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *FriendshipHandler) CancelRequest(c *echo.Context) error {
	user, targetUserID, err := h.authenticatedUserAndPathUserID(c)
	if err != nil {
		return err
	}
	if err := h.usecase.CancelRequest(c.Request().Context(), user.ID, targetUserID); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *FriendshipHandler) Remove(c *echo.Context) error {
	user, friendUserID, err := h.authenticatedUserAndPathUserID(c)
	if err != nil {
		return err
	}
	if err := h.usecase.Remove(c.Request().Context(), user.ID, friendUserID); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *FriendshipHandler) authenticatedUserAndPathUserID(c *echo.Context) (*entity.User, int, error) {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return nil, 0, err
	}
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userID <= 0 {
		return nil, 0, apierror.ErrValidationFailedBadRequest.WithInternal(err)
	}
	return user, userID, nil
}

func toFriendshipListResponse(items []*usecase.FriendshipUserOutput) *internaldto.FriendshipListResponse {
	res := &internaldto.FriendshipListResponse{
		Items: make([]*internaldto.FriendshipUserResponse, 0, len(items)),
	}
	for _, item := range items {
		res.Items = append(res.Items, &internaldto.FriendshipUserResponse{
			UserID:      item.UserID,
			Username:    item.Username,
			PlayerLevel: item.PlayerLevel,
			PlayerName:  item.PlayerName,
			Rating:      item.Rating,
			RequestedAt: item.RequestedAt,
			AcceptedAt:  item.AcceptedAt,
		})
	}
	return res
}
