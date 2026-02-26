package api_internal

import (
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/auth"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// AuthHandler は認証関連のHTTPリクエストを処理します。
type AuthHandler struct {
	authUsecase    usecase.AuthUsecase
	cookieSecure   bool
	cookieSameSite http.SameSite
}

// NewAuthHandler は新しいAuthHandlerを生成します。
func NewAuthHandler(authUsecase usecase.AuthUsecase, cookieSecure bool, cookieSameSite http.SameSite) *AuthHandler {
	return &AuthHandler{
		authUsecase:    authUsecase,
		cookieSecure:   cookieSecure,
		cookieSameSite: cookieSameSite,
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
