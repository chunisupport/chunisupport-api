package api_internal

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type CourseHandler struct{ usecase usecase.CourseUsecase }

func NewCourseHandler(courseUsecase usecase.CourseUsecase) *CourseHandler {
	return &CourseHandler{usecase: courseUsecase}
}

func (h *CourseHandler) List(c *echo.Context) error {
	items, err := h.usecase.List(c.Request().Context(), false)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, &internaldto.CourseListResponse{Courses: items})
}
func (h *CourseHandler) ListEditor(c *echo.Context) error {
	items, err := h.usecase.List(c.Request().Context(), true)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, &internaldto.CourseListResponse{Courses: items})
}
func (h *CourseHandler) Get(c *echo.Context) error {
	item, err := h.usecase.Get(c.Request().Context(), c.Param("idx"), false)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, item)
}
func (h *CourseHandler) GetEditor(c *echo.Context) error {
	item, err := h.usecase.Get(c.Request().Context(), c.Param("idx"), true)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, item)
}
func (h *CourseHandler) Create(c *echo.Context) error {
	var req internaldto.CreateCourseRequest
	if err := handler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	item, err := h.usecase.Create(c.Request().Context(), usecase.CreateCourseInput{Idx: req.Idx, Name: req.Name, Class: req.Class})
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusCreated, item)
}
func (h *CourseHandler) Update(c *echo.Context) error {
	var req internaldto.UpdateCourseRequest
	if err := handler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	item, err := h.usecase.Update(c.Request().Context(), c.Param("idx"), usecase.UpdateCourseInput{Name: req.Name, Class: req.Class})
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, item)
}
func (h *CourseHandler) Delete(c *echo.Context) error {
	if err := h.usecase.Delete(c.Request().Context(), c.Param("idx")); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
func (h *CourseHandler) Restore(c *echo.Context) error {
	if err := h.usecase.Restore(c.Request().Context(), c.Param("idx")); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
func (h *CourseHandler) GetUserRecords(c *echo.Context) error {
	var requester *entity.User
	if v, ok := c.Get("userEntity").(*entity.User); ok {
		requester = v
	}
	include, _ := strconv.ParseBool(c.QueryParam("include_noplay"))
	result, err := h.usecase.GetUserRecords(c.Request().Context(), c.Param("username"), requester, include)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, &internaldto.CourseRecordListResponse{Courses: result.Records, Meta: &internaldto.UserRecordMetaDTO{UpdatedAt: result.UpdatedAt}})
}

func (h *CourseHandler) GetUserRecord(c *echo.Context) error {
	var requester *entity.User
	if value, ok := c.Get("userEntity").(*entity.User); ok {
		requester = value
	}
	result, err := h.usecase.GetUserRecord(c.Request().Context(), c.Param("username"), requester, c.Param("idx"))
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, result)
}
