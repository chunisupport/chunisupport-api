package api_internal

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/auth"
	dto_internal "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// GoalHandler は目標関連のHTTPリクエストを処理します。
type GoalHandler struct {
	goalUsecase usecase.GoalUsecase
}

func NewGoalHandler(goalUsecase usecase.GoalUsecase) *GoalHandler {
	return &GoalHandler{goalUsecase: goalUsecase}
}

func (h *GoalHandler) ListGoals(c echo.Context) error {
	claims, ok := c.Get("user").(*auth.Claims)
	if !ok || claims == nil {
		return apierror.ErrUnauthorized.WithInternal(errors.New("JWT claims not found in context"))
	}
	goals, err := h.goalUsecase.List(c.Request().Context(), claims.UserID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, &dto_internal.GoalListResponseDTO{Goals: goals})
}

func (h *GoalHandler) CreateGoal(c echo.Context) error {
	claims, ok := c.Get("user").(*auth.Claims)
	if !ok || claims == nil {
		return apierror.ErrUnauthorized.WithInternal(errors.New("JWT claims not found in context"))
	}
	var req dto_internal.UpsertGoalRequestDTO
	if err := c.Bind(&req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	created, err := h.goalUsecase.Create(c.Request().Context(), claims.UserID, &req)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusCreated, created)
}

func (h *GoalHandler) UpdateGoal(c echo.Context) error {
	claims, ok := c.Get("user").(*auth.Claims)
	if !ok || claims == nil {
		return apierror.ErrUnauthorized.WithInternal(errors.New("JWT claims not found in context"))
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	var req dto_internal.UpsertGoalRequestDTO
	if err := c.Bind(&req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := h.goalUsecase.Update(c.Request().Context(), claims.UserID, id, &req); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *GoalHandler) DeleteGoal(c echo.Context) error {
	claims, ok := c.Get("user").(*auth.Claims)
	if !ok || claims == nil {
		return apierror.ErrUnauthorized.WithInternal(errors.New("JWT claims not found in context"))
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := h.goalUsecase.Delete(c.Request().Context(), claims.UserID, id); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
