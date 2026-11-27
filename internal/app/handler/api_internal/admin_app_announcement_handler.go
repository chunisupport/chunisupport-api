package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// AdminAppAnnouncementHandler はADMIN専用のお知らせ管理リクエストを処理します。
type AdminAppAnnouncementHandler struct {
	usecase usecase.AdminAppAnnouncementUsecase
}

// NewAdminAppAnnouncementHandler はAdminAppAnnouncementHandlerを生成します。
func NewAdminAppAnnouncementHandler(adminAppAnnouncementUsecase usecase.AdminAppAnnouncementUsecase) *AdminAppAnnouncementHandler {
	return &AdminAppAnnouncementHandler{usecase: adminAppAnnouncementUsecase}
}

// List は終了済み・開始前を含むすべてのお知らせを返します。
func (h *AdminAppAnnouncementHandler) List(c *echo.Context) error {
	announcements, err := h.usecase.ListAll(c.Request().Context())
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, internaldto.ToAppAnnouncementsResponse(announcements))
}

// Create はお知らせを追加します。
func (h *AdminAppAnnouncementHandler) Create(c *echo.Context) error {
	var req internaldto.AppAnnouncementRequest
	if err := apphandler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrValidationFailedBadRequest.WithInternal(err)
	}
	announcement, err := h.usecase.Create(c.Request().Context(), toAppAnnouncementInput(req))
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusCreated, internaldto.ToAppAnnouncementResponse(announcement))
}

// Update はお知らせを上書きします。
func (h *AdminAppAnnouncementHandler) Update(c *echo.Context) error {
	var req internaldto.UpdateAppAnnouncementRequest
	if err := apphandler.BindStrictJSON(c, &req); err != nil {
		return apierror.ErrValidationFailedBadRequest.WithInternal(err)
	}
	announcement, err := h.usecase.Update(c.Request().Context(), c.Param("id"), toAppAnnouncementInput(req.AppAnnouncementRequest), req.BumpRevision)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.JSON(http.StatusOK, internaldto.ToAppAnnouncementResponse(announcement))
}

// Delete はお知らせを削除します。
func (h *AdminAppAnnouncementHandler) Delete(c *echo.Context) error {
	if err := h.usecase.Delete(c.Request().Context(), c.Param("id")); err != nil {
		return apierror.FromUsecaseError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func toAppAnnouncementInput(req internaldto.AppAnnouncementRequest) usecase.AppAnnouncementInput {
	return usecase.AppAnnouncementInput{
		Kind:      req.Kind,
		Title:     req.Title,
		Body:      req.Body,
		TargetURL: req.TargetURL,
		StartsAt:  req.StartsAt,
		ExpiresAt: req.ExpiresAt,
	}
}
