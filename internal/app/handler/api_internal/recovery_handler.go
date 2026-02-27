package api_internal

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// RecoveryHandler はリカバリー関連のリクエストを処理します。
type RecoveryHandler struct {
	recoveryUsecase usecase.RecoveryUsecase
}

// NewRecoveryHandler は新しいRecoveryHandlerを生成します。
func NewRecoveryHandler(recoveryUsecase usecase.RecoveryUsecase) *RecoveryHandler {
	return &RecoveryHandler{recoveryUsecase: recoveryUsecase}
}

type issueRecoveryCodesResponse struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

type recoveryCodeRecoverRequest struct {
	RecoveryCode string `json:"recovery_code"`
	NewPassword  string `json:"new_password"`
}

var recoveryCodeFormat = regexp.MustCompile(`^[A-Za-z0-9]{4}-[A-Za-z0-9]{4}-[A-Za-z0-9]{4}$`)

// IssueRecoveryCodes は認証済みユーザーのリカバリーコードを再発行します。
func (h *RecoveryHandler) IssueRecoveryCodes(c echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	codes, err := h.recoveryUsecase.IssueRecoveryCodes(c.Request().Context(), user.ID)
	if err != nil {
		slog.Error("Failed to issue recovery codes", "user_id", user.ID, "error", err)
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, issueRecoveryCodesResponse{RecoveryCodes: codes})
}

// RecoverPassword はリカバリーコードでパスワードを再設定します。
func (h *RecoveryHandler) RecoverPassword(c echo.Context) error {
	req := new(recoveryCodeRecoverRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if !recoveryCodeFormat.MatchString(req.RecoveryCode) {
		return apierror.ErrBadRequest
	}

	if err := h.recoveryUsecase.RecoverWithRecoveryCode(c.Request().Context(), req.RecoveryCode, req.NewPassword); err != nil {
		slog.Error("Failed to recover password with recovery code", "error", err, "ip_address", c.RealIP())
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusOK)
}
