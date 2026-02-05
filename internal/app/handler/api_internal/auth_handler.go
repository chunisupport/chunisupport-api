package api_internal

import (
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/auth"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/infra/masterdata"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v4"
)

// AuthHandler は認証関連のHTTPリクエストを処理します。
type AuthHandler struct {
	authUsecase    usecase.AuthUsecase
	cookieSecure   bool
	cookieSameSite http.SameSite
	masterCache    *masterdata.Cache
}

const authCookieName = "token"

// NewAuthHandler は新しいAuthHandlerを生成します。
func NewAuthHandler(authUsecase usecase.AuthUsecase, cookieSecure bool, cookieSameSite http.SameSite, masterCache *masterdata.Cache) *AuthHandler {
	return &AuthHandler{
		authUsecase:    authUsecase,
		cookieSecure:   cookieSecure,
		cookieSameSite: cookieSameSite,
		masterCache:    masterCache,
	}
}

// createAuthCookie は認証用のCookieを生成します。
func (h *AuthHandler) createAuthCookie(value string, maxAge int) *http.Cookie {
	cookie := &http.Cookie{
		Name:     authCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: h.cookieSameSite,
		MaxAge:   maxAge,
	}
	if maxAge < 0 {
		cookie.Expires = time.Unix(0, 0).UTC()
	}
	return cookie
}

// getUserEntity はコンテキストからユーザーエンティティを取得します。
func (h *AuthHandler) getUserEntity(c echo.Context) (*entity.User, error) {
	user, ok := c.Get("userEntity").(*entity.User)
	if !ok {
		return nil, apierror.ErrUnauthorized
	}
	return user, nil
}

// authRequest は認証リクエストのボディの構造です。
type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
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

	c.SetCookie(h.createAuthCookie(token, 0))

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

	c.SetCookie(h.createAuthCookie(token, 0))

	return c.NoContent(http.StatusOK)
}

// Logout はログアウトリクエストを処理します。
func (h *AuthHandler) Logout(c echo.Context) error {
	claims, ok := c.Get("user").(*auth.Claims)
	if !ok || claims == nil {
		// ミドルウェアでセットされているはずなので、ここに来ることは基本ない
		return apierror.ErrUnauthorized
	}

	if err := h.authUsecase.Logout(c.Request().Context(), claims.SessionID); err != nil {
		slog.Error("Failed to logout", "session_id", claims.SessionID, "error", err.Error())
		return apierror.ErrInternalError.WithInternal(err)
	}

	c.SetCookie(h.createAuthCookie("", -1))

	return c.NoContent(http.StatusOK)
}

// Me は認証済みユーザー自身の情報を取得するリクエストを処理します。
func (h *AuthHandler) Me(c echo.Context) error {
	user, err := h.getUserEntity(c)
	if err != nil {
		return err
	}

	userDTO, err := h.authUsecase.GetUser(c.Request().Context(), user.ID)
	if err != nil {
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, userDTO)
}

// updatePrivacyRequest はプライバシー設定更新リクエストのボディの構造です。
type updatePrivacyRequest struct {
	IsPrivate bool `json:"is_private"`
}

// UpdatePrivacy は認証済みユーザーの非公開設定を更新するリクエストを処理します。
func (h *AuthHandler) UpdatePrivacy(c echo.Context) error {
	user, err := h.getUserEntity(c)
	if err != nil {
		return err
	}

	req := new(updatePrivacyRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	if err := h.authUsecase.UpdatePrivacy(c.Request().Context(), user.ID, req.IsPrivate); err != nil {
		slog.Error("Failed to update privacy setting", "user_id", user.ID, "error", err.Error())
		return apierror.ErrInternalError.WithInternal(err)
	}

	// 更新された設定を反映
	user.IsPrivate = req.IsPrivate

	return c.JSON(http.StatusOK, map[string]any{
		"is_private": req.IsPrivate,
	})
}

// changePasswordRequest はパスワード変更リクエストのボディの構造です。
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type issueRecoveryCodesResponse struct {
	RecoveryCodes []string `json:"recovery_codes"`
}

type recoveryCodeRecoverRequest struct {
	RecoveryCode string `json:"recovery_code"`
	NewPassword  string `json:"new_password"`
}

var recoveryCodeFormat = regexp.MustCompile(`^[A-Za-z0-9]{4}-[A-Za-z0-9]{4}-[A-Za-z0-9]{4}$`)

// ChangePassword は認証済みユーザーのパスワードを変更するリクエストを処理します。
func (h *AuthHandler) ChangePassword(c echo.Context) error {
	user, err := h.getUserEntity(c)
	if err != nil {
		return err
	}

	req := new(changePasswordRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}

	if err := h.authUsecase.ChangePassword(c.Request().Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		slog.Error("Failed to change password", "user_id", user.ID, "error", err.Error())
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusOK)
}

// IssueRecoveryCodes は認証済みユーザーのリカバリーコードを再発行します。
func (h *AuthHandler) IssueRecoveryCodes(c echo.Context) error {
	user, err := h.getUserEntity(c)
	if err != nil {
		return err
	}

	codes, err := h.authUsecase.IssueRecoveryCodes(c.Request().Context(), user.ID)
	if err != nil {
		slog.Error("Failed to issue recovery codes", "user_id", user.ID, "error", err.Error())
		return apierror.FromUsecaseError(err)
	}

	return c.JSON(http.StatusOK, issueRecoveryCodesResponse{
		RecoveryCodes: codes,
	})
}

// RecoverPassword はリカバリーコードでパスワードを再設定します。
func (h *AuthHandler) RecoverPassword(c echo.Context) error {
	req := new(recoveryCodeRecoverRequest)
	if err := c.Bind(req); err != nil {
		return apierror.ErrBadRequest.WithInternal(err)
	}
	if !recoveryCodeFormat.MatchString(req.RecoveryCode) {
		return apierror.ErrBadRequest
	}

	if err := h.authUsecase.RecoverWithRecoveryCode(c.Request().Context(), req.RecoveryCode, req.NewPassword); err != nil {
		slog.Error("Failed to recover password with recovery code", "error", err.Error(), "ip_address", c.RealIP())
		return apierror.FromUsecaseError(err)
	}

	return c.NoContent(http.StatusOK)
}

// DeleteAccount は認証済みユーザーの論理削除を行うリクエストを処理します。
func (h *AuthHandler) DeleteAccount(c echo.Context) error {
	user, err := h.getUserEntity(c)
	if err != nil {
		return err
	}

	if err := h.authUsecase.DeleteUser(c.Request().Context(), user.ID); err != nil {
		slog.Error("Failed to delete user", "user_id", user.ID, "error", err.Error())
		return apierror.ErrInternalError.WithInternal(err)
	}

	if claims, ok := c.Get("user").(*auth.Claims); ok && claims != nil {
		if err := h.authUsecase.Logout(c.Request().Context(), claims.SessionID); err != nil {
			slog.Error("Failed to invalidate session after deletion", "session_id", claims.SessionID, "error", err.Error())
			return apierror.ErrInternalError.WithInternal(err)
		}
	}

	c.SetCookie(h.createAuthCookie("", -1))
	user.IsDeleted = true

	return c.NoContent(http.StatusOK)
}
