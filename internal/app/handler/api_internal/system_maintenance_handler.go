package api_internal

import (
	"context"
	"net/http"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	apphandler "github.com/chunisupport/chunisupport-api/internal/app/handler"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

type systemMaintenanceUsecase interface {
	usecase.MaintenanceStateProvider
	Update(ctx context.Context, actorUserID int, enabled bool, comment string) (usecase.MaintenanceState, error)
}

type systemMaintenanceRequest struct {
	Enabled *bool  `json:"enabled"`
	Comment string `json:"comment"`
}

type systemMaintenanceResponse struct {
	Status    string    `json:"status"`
	Comment   string    `json:"comment"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SystemMaintenanceHandler は公開状態確認とADMINによる状態変更を処理します。
type SystemMaintenanceHandler struct {
	usecase systemMaintenanceUsecase
}

// NewSystemMaintenanceHandler はSystemMaintenanceHandlerを生成します。
func NewSystemMaintenanceHandler(maintenanceUsecase systemMaintenanceUsecase) *SystemMaintenanceHandler {
	return &SystemMaintenanceHandler{usecase: maintenanceUsecase}
}

// Status は現在の運用状態を認証不要で返します。
func (h *SystemMaintenanceHandler) Status(c *echo.Context) error {
	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.JSON(http.StatusOK, toSystemMaintenanceResponse(h.usecase.Current()))
}

// Update はADMINが指定したメンテナンス状態を保存します。
func (h *SystemMaintenanceHandler) Update(c *echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	var request systemMaintenanceRequest
	if err := apphandler.BindStrictJSON(c, &request); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if request.Enabled == nil {
		return apierror.ErrBadRequest
	}

	state, err := h.usecase.Update(
		c.Request().Context(),
		user.ID,
		*request.Enabled,
		request.Comment,
	)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.Response().Header().Set(echo.HeaderCacheControl, "no-store")
	return c.JSON(http.StatusOK, toSystemMaintenanceResponse(state))
}

func toSystemMaintenanceResponse(state usecase.MaintenanceState) *systemMaintenanceResponse {
	status := systemStatusOperational
	if state.Enabled {
		status = systemStatusMaintenance
	}

	return &systemMaintenanceResponse{
		Status:    status,
		Comment:   state.Comment,
		UpdatedAt: state.UpdatedAt,
	}
}

var _ usecase.MaintenanceStateProvider = (systemMaintenanceUsecase)(nil)
