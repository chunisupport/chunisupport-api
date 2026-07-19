package api_internal

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// GoalGroupHandler は目標グループAPIを扱います。
type GoalGroupHandler struct {
	usecase usecase.GoalGroupUsecase
}

// NewGoalGroupHandler は目標グループハンドラーを生成します。
func NewGoalGroupHandler(goalGroupUsecase usecase.GoalGroupUsecase) *GoalGroupHandler {
	return &GoalGroupHandler{usecase: goalGroupUsecase}
}

func (h *GoalGroupHandler) List(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	groups, err := h.usecase.List(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	items := make([]*internaldto.GoalGroupResponse, 0, len(groups))
	for _, group := range groups {
		items = append(items, toGoalGroupResponse(group))
	}
	return c.JSON(http.StatusOK, &internaldto.GoalGroupsResponse{Groups: items})
}

func (h *GoalGroupHandler) Create(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	var request internaldto.GoalGroupRequest
	if err := apphandler.BindStrictJSON(c, &request); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&request); err != nil {
		return err
	}
	group, err := h.usecase.Create(c.Request().Context(), user.ID, request.Name)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusCreated, toGoalGroupResponse(group))
}

func (h *GoalGroupHandler) Update(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	id, err := parseGoalGroupID(c)
	if err != nil {
		return err
	}
	var request internaldto.GoalGroupRequest
	if err := apphandler.BindStrictJSON(c, &request); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&request); err != nil {
		return err
	}
	group, err := h.usecase.Update(c.Request().Context(), user.ID, id, request.Name)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toGoalGroupResponse(group))
}

func (h *GoalGroupHandler) Delete(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	id, err := parseGoalGroupID(c)
	if err != nil {
		return err
	}
	if err := h.usecase.Delete(c.Request().Context(), user.ID, id); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *GoalGroupHandler) Reorder(c *echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	var request internaldto.GoalGroupOrderRequest
	if err := apphandler.BindStrictJSON(c, &request); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if request.GroupIDs == nil {
		return apierror.ErrBadRequest
	}
	if err := h.usecase.Reorder(c.Request().Context(), user.ID, request.GroupIDs); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func parseGoalGroupID(c *echo.Context) (uint32, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return 0, apierror.ErrBadRequest.WithInternal(err)
	}
	return uint32(id), nil
}

func toGoalGroupResponse(group *usecase.GoalGroupOutput) *internaldto.GoalGroupResponse {
	return &internaldto.GoalGroupResponse{ID: group.ID, Name: group.Name, SortOrder: group.SortOrder, CreatedAt: group.CreatedAt}
}
