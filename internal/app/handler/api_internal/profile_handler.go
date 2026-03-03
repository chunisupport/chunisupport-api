package api_internal

import (
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/auth"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// ProfileHandler は認証済みユーザーのプロフィール関連リクエストを処理します。
type ProfileHandler struct {
	authUsecase           usecase.AuthUsecase
	userCredentialUsecase usecase.UserCredentialUsecase
	cookieSecure          bool
	cookieSameSite        http.SameSite
}

// NewProfileHandler は新しいProfileHandlerを生成します。
func NewProfileHandler(authUsecase usecase.AuthUsecase, userCredentialUsecase usecase.UserCredentialUsecase, cookieSecure bool, cookieSameSite http.SameSite) *ProfileHandler {
	return &ProfileHandler{
		authUsecase:           authUsecase,
		userCredentialUsecase: userCredentialUsecase,
		cookieSecure:          cookieSecure,
		cookieSameSite:        cookieSameSite,
	}
}

// Me は認証済みユーザー自身の情報を取得するリクエストを処理します。
func (h *ProfileHandler) Me(c echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	userDTO, err := h.userCredentialUsecase.GetUser(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, userDTO)
}

type updatePrivacyRequest struct {
	IsPrivate bool `json:"is_private"`
}

// UpdatePrivacy は認証済みユーザーの非公開設定を更新するリクエストを処理します。
func (h *ProfileHandler) UpdatePrivacy(c echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	req := new(updatePrivacyRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	if err := h.userCredentialUsecase.UpdatePrivacy(c.Request().Context(), user.ID, req.IsPrivate); err != nil {
		slog.Error("Failed to update privacy setting", "user_id", user.ID, "error", err)
		return apierror.ErrInternalError.WithInternal(err)
	}

	return c.JSON(http.StatusOK, map[string]any{"is_private": req.IsPrivate})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required,min=8,max=128"`
	NewPassword     string `json:"new_password" validate:"required,min=8,max=128"`
}

// ChangePassword は認証済みユーザーのパスワードを変更するリクエストを処理します。
func (h *ProfileHandler) ChangePassword(c echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	req := new(changePasswordRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if err := c.Validate(req); err != nil {
		return err
	}

	if err := h.userCredentialUsecase.ChangePassword(c.Request().Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		slog.Error("Failed to change password", "user_id", user.ID, "error", err)
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusOK)
}

// DeleteAccount は認証済みユーザーの論理削除を行うリクエストを処理します。
func (h *ProfileHandler) DeleteAccount(c echo.Context) error {
	user, err := getUserEntityFromContext(c)
	if err != nil {
		return err
	}

	if err := h.userCredentialUsecase.DeleteOwnAccount(c.Request().Context(), user.ID); err != nil {
		slog.Error("Failed to delete user", "user_id", user.ID, "error", err)
		return apierror.FromUsecaseError(err)
	}

	if claims, ok := c.Get("user").(*auth.Claims); ok && claims != nil {
		if err := h.authUsecase.Logout(c.Request().Context(), claims.SessionID); err != nil {
			slog.Error("Failed to invalidate session after deletion", "session_id", claims.SessionID, "error", err)
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	c.SetCookie(newAuthCookie(h.cookieSecure, h.cookieSameSite, "", -1))

	return c.NoContent(http.StatusOK)
}
