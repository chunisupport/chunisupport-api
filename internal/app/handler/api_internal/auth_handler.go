package api_internal

import (
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/auth"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// AuthHandler は認証関連のHTTPリクエストを処理します。
type AuthHandler struct {
	authUsecase           usecase.AuthUsecase
	userCredentialUsecase usecase.UserCredentialUsecase
	recoveryUsecase       usecase.RecoveryUsecase
	cookieSecure          bool
	cookieSameSite        http.SameSite
	masterCache           *masterdata.Cache
}

// NewAuthHandler は新しいAuthHandlerを生成します。
func NewAuthHandler(authUsecase usecase.AuthUsecase, userCredentialUsecase usecase.UserCredentialUsecase, recoveryUsecase usecase.RecoveryUsecase, cookieSecure bool, cookieSameSite http.SameSite, masterCache *masterdata.Cache) *AuthHandler {
	return &AuthHandler{
		authUsecase:           authUsecase,
		userCredentialUsecase: userCredentialUsecase,
		recoveryUsecase:       recoveryUsecase,
		cookieSecure:          cookieSecure,
		cookieSameSite:        cookieSameSite,
		masterCache:           masterCache,
	}
}

// authRequest は認証リクエストのボディの構造です。
type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"` // #nosec G117 API入力仕様として必要
}

// Register はユーザー登録リクエストを処理します。
// 登録成功時は自動的にログイン状態となり、認証Cookieが設定されます。
func (h *AuthHandler) Register(c echo.Context) error {
	req := new(authRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	user, token, err := h.authUsecase.Register(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.SetCookie(newAuthCookie(h.cookieSecure, h.cookieSameSite, token, 0))

	return c.JSON(http.StatusCreated, user)
}

// Login はログインリクエストを処理します。
func (h *AuthHandler) Login(c echo.Context) error {
	req := new(authRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	token, err := h.authUsecase.Login(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	c.SetCookie(newAuthCookie(h.cookieSecure, h.cookieSameSite, token, 0))

	return c.NoContent(http.StatusOK)
}

// Logout はログアウトリクエストを処理します。
func (h *AuthHandler) Logout(c echo.Context) error {
	claims, ok := c.Get("user").(*auth.Claims)
	if !ok || claims == nil {
		return apierror.ErrUnauthorized
	}

	if err := h.authUsecase.Logout(c.Request().Context(), claims.SessionID); err != nil {
		slog.Error("Failed to logout", "session_id", claims.SessionID, "error", err)
		return apierror.ErrInternalError.WithInternal(err)
	}

	c.SetCookie(newAuthCookie(h.cookieSecure, h.cookieSameSite, "", -1))

	return c.NoContent(http.StatusOK)
}

// Me は後方互換のため残した委譲メソッドです。
func (h *AuthHandler) Me(c echo.Context) error {
	profileHandler := NewProfileHandler(h.authUsecase, h.userCredentialUsecase, h.cookieSecure, h.cookieSameSite)
	return profileHandler.Me(c)
}

// UpdatePrivacy は後方互換のため残した委譲メソッドです。
func (h *AuthHandler) UpdatePrivacy(c echo.Context) error {
	profileHandler := NewProfileHandler(h.authUsecase, h.userCredentialUsecase, h.cookieSecure, h.cookieSameSite)
	return profileHandler.UpdatePrivacy(c)
}

// ChangePassword は後方互換のため残した委譲メソッドです。
func (h *AuthHandler) ChangePassword(c echo.Context) error {
	profileHandler := NewProfileHandler(h.authUsecase, h.userCredentialUsecase, h.cookieSecure, h.cookieSameSite)
	return profileHandler.ChangePassword(c)
}

// DeleteAccount は後方互換のため残した委譲メソッドです。
func (h *AuthHandler) DeleteAccount(c echo.Context) error {
	profileHandler := NewProfileHandler(h.authUsecase, h.userCredentialUsecase, h.cookieSecure, h.cookieSameSite)
	return profileHandler.DeleteAccount(c)
}

// IssueRecoveryCodes は後方互換のため残した委譲メソッドです。
func (h *AuthHandler) IssueRecoveryCodes(c echo.Context) error {
	recoveryHandler := NewRecoveryHandler(h.recoveryUsecase)
	return recoveryHandler.IssueRecoveryCodes(c)
}

// RecoverPassword は後方互換のため残した委譲メソッドです。
func (h *AuthHandler) RecoverPassword(c echo.Context) error {
	recoveryHandler := NewRecoveryHandler(h.recoveryUsecase)
	return recoveryHandler.RecoverPassword(c)
}
