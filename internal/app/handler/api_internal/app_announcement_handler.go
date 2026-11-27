package api_internal

import (
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	internaldto "github.com/chunisupport/chunisupport-api/internal/dto/api_internal"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// AppAnnouncementHandler は期間限定のお知らせを返します。
type AppAnnouncementHandler struct {
	usecase usecase.AppAnnouncementUsecase
}

// NewAppAnnouncementHandler はAppAnnouncementHandlerを生成します。
func NewAppAnnouncementHandler(appAnnouncementUsecase usecase.AppAnnouncementUsecase) *AppAnnouncementHandler {
	return &AppAnnouncementHandler{usecase: appAnnouncementUsecase}
}

// List は現在配信中のお知らせを返します。
// 配信開始・終了・緊急停止を即時に反映するため、キャッシュさせません。
func (h *AppAnnouncementHandler) List(c *echo.Context) error {
	announcements, err := h.usecase.ListActive(c.Request().Context())
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.JSON(http.StatusOK, internaldto.ToAppAnnouncementsResponse(announcements))
}
