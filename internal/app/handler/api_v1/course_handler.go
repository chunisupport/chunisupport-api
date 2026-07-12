package api_v1

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
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
		result = append(result, api_v1.ToV1CourseDTO(item))
	}
	return c.JSON(http.StatusOK, &api_v1.V1CourseListResponse{Courses: result})
}
func (h *V1CourseHandler) Get(c *echo.Context) error {
	item, err := h.usecase.Get(c.Request().Context(), c.Param("idx"), false)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, api_v1.ToV1CourseDTO(item))
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
	return c.JSON(http.StatusOK, &api_v1.V1CourseRecordListResponse{UpdatedAt: result.UpdatedAt, Courses: result.Records})
}
