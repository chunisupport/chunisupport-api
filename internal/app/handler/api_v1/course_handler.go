package api_v1

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_v1"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type V1CourseHandler struct{ usecase usecase.CourseUsecase }

func NewV1CourseHandler(courseUsecase usecase.CourseUsecase) *V1CourseHandler {
	return &V1CourseHandler{usecase: courseUsecase}
}
func (h *V1CourseHandler) List(c *echo.Context) error {
	items, err := h.usecase.List(c.Request().Context(), false)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	result := make([]*api_v1.V1CourseDTO, 0, len(items))
	for _, item := range items {
		result = append(result, toV1CourseDTO(item))
	}
	return c.JSON(http.StatusOK, &api_v1.V1CourseListResponse{Courses: result})
}
func (h *V1CourseHandler) Get(c *echo.Context) error {
	displayID, apiErr := handler.ValidateDisplayID(c.Param("id"))
	if apiErr != nil {
		return apiErr
	}
	item, err := h.usecase.Get(c.Request().Context(), displayID, false)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toV1CourseDTO(item))
}
func (h *V1CourseHandler) GetUserRecords(c *echo.Context) error {
	var requester *entity.User
	if v, ok := c.Get("userEntity").(*entity.User); ok {
		requester = v
	}
	include, _ := strconv.ParseBool(c.QueryParam("include_noplay"))
	result, err := h.usecase.GetUserRecords(c.Request().Context(), c.Param("username"), requester, include)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, &api_v1.V1CourseRecordListResponse{UpdatedAt: result.UpdatedAt, Courses: toV1CourseRecordDTOs(result.Records)})
}

func toV1CourseDTO(value *usecase.CourseOutput) *api_v1.V1CourseDTO {
	if value == nil {
		return nil
	}
	return &api_v1.V1CourseDTO{DisplayID: value.DisplayID, Idx: value.Idx, Name: value.Name, Class: value.Class}
}
func toV1CourseRecordDTOs(values []*usecase.CourseRecordOutput) []*api_v1.V1CourseRecordDTO {
	result := make([]*api_v1.V1CourseRecordDTO, 0, len(values))
	for _, value := range values {
		if value == nil {
			result = append(result, nil)
			continue
		}
		result = append(result, &api_v1.V1CourseRecordDTO{DisplayID: value.DisplayID, Idx: value.Idx, Name: value.Name, Class: value.Class, IsPlayed: value.IsPlayed, Score: value.Score, IsClear: value.IsClear, ComboLamp: value.ComboLamp, UpdatedAt: value.UpdatedAt})
	}
	return result
}
