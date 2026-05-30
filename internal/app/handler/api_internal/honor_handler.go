package api_internal

import (
	"net/http"
	"strconv"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// HonorHandler はADMIN専用の称号関連HTTPリクエストを処理します。
type HonorHandler struct {
	honorUsecase usecase.HonorUsecase
}

// NewHonorHandler は新しい HonorHandler を生成します。
func NewHonorHandler(honorUsecase usecase.HonorUsecase) *HonorHandler {
	return &HonorHandler{honorUsecase: honorUsecase}
}

// ListHonors は称号一覧を返却します。
func (h *HonorHandler) ListHonors(c echo.Context) error {
	honors, err := h.honorUsecase.ListHonors(c.Request().Context())
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, &api_internal.HonorsResponse{
		Honors: api_internal.ToHonorDTOs(honors),
	})
}

// GetHonor は指定IDの称号を返却します。
func (h *HonorHandler) GetHonor(c echo.Context) error {
	id, err := parseHonorID(c.Param("id"))
	if err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}

	honor, err := h.honorUsecase.GetHonor(c.Request().Context(), id)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, api_internal.ToHonorDTO(honor))
}

// CreateHonor は称号を作成します。
func (h *HonorHandler) CreateHonor(c echo.Context) error {
	var req api_internal.HonorRequest
	if err := c.Bind(&req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}

	honor, err := h.honorUsecase.CreateHonor(c.Request().Context(), usecase.HonorInput{
		Name:     req.Name,
		TypeName: req.TypeName,
		ImageURL: req.ImageURL,
	})
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusCreated, api_internal.ToHonorDTO(honor))
}

// UpdateHonor は称号を更新します。
func (h *HonorHandler) UpdateHonor(c echo.Context) error {
	id, err := parseHonorID(c.Param("id"))
	if err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}

	var req api_internal.HonorRequest
	if err := c.Bind(&req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(&req); err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}

	honor, err := h.honorUsecase.UpdateHonor(c.Request().Context(), id, usecase.HonorInput{
		Name:     req.Name,
		TypeName: req.TypeName,
		ImageURL: req.ImageURL,
	})
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, api_internal.ToHonorDTO(honor))
}

// DeleteHonor は称号を削除します。
func (h *HonorHandler) DeleteHonor(c echo.Context) error {
	id, err := parseHonorID(c.Param("id"))
	if err != nil {
		return apierror.ErrValidationFailed.WithInternal(err)
	}

	if err := h.honorUsecase.DeleteHonor(c.Request().Context(), id); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func parseHonorID(raw string) (int, error) {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, usecase.ErrInvalidHonorInput
	}
	return id, nil
}
