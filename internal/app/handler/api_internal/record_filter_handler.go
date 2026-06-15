package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// RecordFilterHandler は保存済み譜面フィルタAPIを扱います。
type RecordFilterHandler struct {
	recordFilterUsecase usecase.RecordFilterUsecase
}

// NewRecordFilterHandler は RecordFilterHandler を生成します。
func NewRecordFilterHandler(recordFilterUsecase usecase.RecordFilterUsecase) *RecordFilterHandler {
	return &RecordFilterHandler{recordFilterUsecase: recordFilterUsecase}
}

func (h *RecordFilterHandler) List(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	filters, err := h.recordFilterUsecase.List(c.Request().Context(), user.ID, c.QueryParam("filter_type"))
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	items := make([]*internaldto.RecordFilterResponse, 0, len(filters))
	for _, filter := range filters {
		items = append(items, toRecordFilterResponse(filter))
	}
	return c.JSON(http.StatusOK, &internaldto.RecordFiltersResponse{Filters: items})
}

func (h *RecordFilterHandler) Create(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}

	var req internaldto.RecordFilterRequest
	if err := apphandler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	filter, err := h.recordFilterUsecase.Create(c.Request().Context(), user.ID, toRecordFilterInput(&req))
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusCreated, toRecordFilterResponse(filter))
}

func (h *RecordFilterHandler) Update(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}

	var req internaldto.RecordFilterRequest
	if err := apphandler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	filter, err := h.recordFilterUsecase.Update(c.Request().Context(), user.ID, c.Param("id"), toRecordFilterInput(&req))
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, toRecordFilterResponse(filter))
}

func (h *RecordFilterHandler) Delete(c echo.Context) error {
	user, err := getUser(c)
	if err != nil {
		return err
	}
	if err := h.recordFilterUsecase.Delete(c.Request().Context(), user.ID, c.Param("id")); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func toRecordFilterInput(req *internaldto.RecordFilterRequest) *usecase.RecordFilterInput {
	return &usecase.RecordFilterInput{
		Name:          req.Name,
		FilterType:    req.FilterType,
		SchemaVersion: req.SchemaVersion,
		Filter:        req.Filter,
	}
}

func toRecordFilterResponse(filter *usecase.RecordFilterOutput) *internaldto.RecordFilterResponse {
	return &internaldto.RecordFilterResponse{
		ID:            filter.ID,
		Name:          filter.Name,
		FilterType:    filter.FilterType,
		SchemaVersion: filter.SchemaVersion,
		Filter:        filter.Filter,
		CreatedAt:     filter.CreatedAt,
		UpdatedAt:     filter.UpdatedAt,
	}
}
