package api_internal

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// VersionHandler はADMIN専用のバージョン管理リクエストを処理します。
type VersionHandler struct {
	versionUsecase usecase.VersionUsecase
}

// NewVersionHandler はVersionHandlerを生成します。
func NewVersionHandler(versionUsecase usecase.VersionUsecase) *VersionHandler {
	return &VersionHandler{versionUsecase: versionUsecase}
}

func (h *VersionHandler) List(c *echo.Context) error {
	versions, err := h.versionUsecase.ListAll(c.Request().Context())
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, internaldto.ToVersionDTOs(versions))
}

func (h *VersionHandler) Create(c *echo.Context) error {
	var req internaldto.CreateVersionRequest
	if err := c.Bind(&req); err != nil {
		return apierror.ErrValidationFailedBadRequest.WithInternal(err)
	}
	releasedAt, err := time.Parse(time.DateOnly, req.ReleasedAt)
	if err != nil {
		return apierror.FromUsecaseError(errors.Join(usecase.ErrInvalidVersionInput, err))
	}
	version, err := h.versionUsecase.Create(c.Request().Context(), req.Name, releasedAt)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusCreated, internaldto.ToVersionDTO(version))
}

func (h *VersionHandler) Rename(c *echo.Context) error {
	id, err := parseVersionID(c.Param("id"))
	if err != nil {
		return apierror.ErrValidationFailedBadRequest.WithInternal(err)
	}
	var req internaldto.RenameVersionRequest
	if err := c.Bind(&req); err != nil {
		return apierror.ErrValidationFailedBadRequest.WithInternal(err)
	}
	if req.ReleasedAt != nil {
		return apierror.ErrValidationFailedBadRequest.WithInternal(errors.New("released_at cannot be changed"))
	}
	version, err := h.versionUsecase.Rename(c.Request().Context(), id, req.Name)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, internaldto.ToVersionDTO(version))
}

func (h *VersionHandler) Delete(c *echo.Context) error {
	id, err := parseVersionID(c.Param("id"))
	if err != nil {
		return apierror.ErrValidationFailedBadRequest.WithInternal(err)
	}
	if err := h.versionUsecase.Delete(c.Request().Context(), id); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func parseVersionID(raw string) (int, error) {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, usecase.ErrInvalidVersionInput
	}
	return id, nil
}
