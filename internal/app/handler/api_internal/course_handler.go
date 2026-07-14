package api_internal

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto"
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
	return c.JSON(http.StatusOK, &internaldto.CourseListResponse{Courses: toCourseDTOs(items)})
}

// GetCoursesUpdatedAt はコースマスタの updated_at のみを返します。
func (h *CourseHandler) GetCoursesUpdatedAt(c *echo.Context) error {
	updatedAt, err := h.usecase.GetCoursesUpdatedAt(c.Request().Context())
	if err != nil {
		return apierror.ErrInternalError.WithInternal(err)
	}
	return c.JSON(http.StatusOK, &internaldto.CourseUpdatedAtDTO{UpdatedAt: updatedAt})
}
func (h *CourseHandler) ListEditor(c *echo.Context) error {
	items, err := h.usecase.List(c.Request().Context(), true)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, &internaldto.CourseListResponse{Courses: toCourseDTOs(items)})
}
func (h *CourseHandler) Get(c *echo.Context) error {
	displayID, apiErr := handler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	item, err := h.usecase.Get(c.Request().Context(), displayID, false)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toCourseDTO(item))
}
func (h *CourseHandler) GetEditor(c *echo.Context) error {
	displayID, apiErr := handler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	item, err := h.usecase.Get(c.Request().Context(), displayID, true)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toCourseDTO(item))
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
	return c.JSON(http.StatusCreated, toCourseDTO(item))
}
func (h *CourseHandler) Update(c *echo.Context) error {
	displayID, apiErr := handler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	var req internaldto.UpdateCourseRequest
	if err := handler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}
	item, err := h.usecase.Update(c.Request().Context(), displayID, usecase.UpdateCourseInput{Name: req.Name, Class: req.Class})
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toCourseDTO(item))
}
func (h *CourseHandler) Delete(c *echo.Context) error {
	displayID, apiErr := handler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	if err := h.usecase.Delete(c.Request().Context(), displayID); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
func (h *CourseHandler) Restore(c *echo.Context) error {
	displayID, apiErr := handler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	if err := h.usecase.Restore(c.Request().Context(), displayID); err != nil {
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
	return c.JSON(http.StatusOK, &internaldto.CourseRecordListResponse{Courses: toCourseRecordDTOs(result.Records), Meta: &internaldto.UserRecordMetaDTO{UpdatedAt: result.UpdatedAt}})
}

func (h *CourseHandler) GetUserRecord(c *echo.Context) error {
	displayID, apiErr := handler.ValidateDisplayID(c.Param("displayid"))
	if apiErr != nil {
		return apiErr
	}
	var requester *entity.User
	if value, ok := c.Get("userEntity").(*entity.User); ok {
		requester = value
	}
	result, err := h.usecase.GetUserRecord(c.Request().Context(), c.Param("username"), requester, displayID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toCourseRecordDTO(result))
}

func toCourseDTO(value *usecase.CourseOutput) *dto.CourseDTO {
	if value == nil {
		return nil
	}
	return &dto.CourseDTO{ID: value.ID, DisplayID: value.DisplayID, Idx: value.Idx, Name: value.Name, Class: value.Class, IsDeleted: value.IsDeleted, UpdatedAt: value.UpdatedAt}
}
func toCourseDTOs(values []*usecase.CourseOutput) []*dto.CourseDTO {
	result := make([]*dto.CourseDTO, 0, len(values))
	for _, value := range values {
		result = append(result, toCourseDTO(value))
	}
	return result
}
func toCourseRecordDTO(value *usecase.CourseRecordOutput) *dto.CourseRecordDTO {
	if value == nil {
		return nil
	}
	return &dto.CourseRecordDTO{DisplayID: value.DisplayID, Idx: value.Idx, Name: value.Name, Class: value.Class, IsPlayed: value.IsPlayed, Score: value.Score, IsClear: value.IsClear, ComboLamp: value.ComboLamp, UpdatedAt: value.UpdatedAt}
}
func toCourseRecordDTOs(values []*usecase.CourseRecordOutput) []*dto.CourseRecordDTO {
	result := make([]*dto.CourseRecordDTO, 0, len(values))
	for _, value := range values {
		result = append(result, toCourseRecordDTO(value))
	}
	return result
}
