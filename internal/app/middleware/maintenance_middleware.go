package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/labstack/echo/v5"
)

// APITokenValidator はAPIトークンから利用者とトークン情報を解決します。
type APITokenValidator interface {
	Validate(ctx context.Context, rawToken string) (*entity.User, *entity.APIToken, error)
}

// FirebaseMaintenanceMiddleware はメンテナンス中のinternal APIをスタッフだけに制限します。
// 通常時は状態スナップショットの参照だけで通過し、Firebase認証を追加実行しません。
func FirebaseMaintenanceMiddleware(
	stateProvider usecase.MaintenanceStateProvider,
	authenticator FirebaseAuthenticator,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if stateProvider == nil {
				return apierror.ErrInternalError.WithInternal(errors.New("maintenance state provider is nil"))
			}
			if !stateProvider.Current().Enabled || shouldBypassMaintenanceGate(c) {
				return next(c)
			}

			idToken := extractBearerToken(c)
			if idToken == "" {
				return maintenanceModeError(c)
			}
			if authenticator == nil {
				slog.Error("Maintenance Firebase authentication failed", "error", "firebase authenticator is nil")
				return maintenanceModeError(c)
			}

			user, err := authenticator.Authenticate(c.Request().Context(), idToken)
			if err != nil {
				if !errors.Is(err, usecase.ErrInvalidIDToken) {
					slog.Error("Maintenance Firebase authentication failed", "error", err)
				}
				return maintenanceModeError(c)
			}
			if user == nil {
				slog.Error("Maintenance Firebase authentication failed", "error", "firebase authenticator returned nil user")
				return maintenanceModeError(c)
			}
			if !info.HasRole(user.AccountTypeID, info.AccountTypeEditor) {
				return maintenanceModeError(c)
			}

			c.Set(contextKeyUserEntity, user)
			return next(c)
		}
	}
}

// APITokenMaintenanceMiddleware はメンテナンス中の外部APIをスタッフだけに制限します。
// 解決した認証情報は後続の通常認証で再利用されるため、同じトークンを二重検証しません。
func APITokenMaintenanceMiddleware(
	stateProvider usecase.MaintenanceStateProvider,
	validator APITokenValidator,
) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if stateProvider == nil {
				return apierror.ErrInternalError.WithInternal(errors.New("maintenance state provider is nil"))
			}
			if !stateProvider.Current().Enabled || shouldBypassMaintenanceGate(c) {
				return next(c)
			}

			rawToken := extractBearerToken(c)
			if rawToken == "" {
				return maintenanceModeError(c)
			}
			if validator == nil {
				slog.Error("Maintenance API token authentication failed", "error", "API token validator is nil")
				return maintenanceModeError(c)
			}

			user, token, err := validator.Validate(c.Request().Context(), rawToken)
			if err != nil {
				if !errors.Is(err, usecase.ErrInvalidAPIToken) {
					slog.Error("Maintenance API token authentication failed", "error", err)
				}
				return maintenanceModeError(c)
			}
			if user == nil || token == nil {
				slog.Error("Maintenance API token authentication failed", "error", "API token validator returned nil authentication data")
				return maintenanceModeError(c)
			}
			if !info.HasRole(user.AccountTypeID, info.AccountTypeEditor) {
				return maintenanceModeError(c)
			}

			c.Set(contextKeyUserEntity, user)
			c.Set(contextKeyAPIToken, token)
			return next(c)
		}
	}
}

func shouldBypassMaintenanceGate(c *echo.Context) bool {
	method := c.Request().Method
	path := c.Request().URL.Path
	if method == http.MethodOptions {
		return true
	}

	return (method == http.MethodGet && (path == "/healthz" || path == "/internal/system/status")) ||
		(method == http.MethodPost && path == "/internal/auth/login")
}

func maintenanceModeError(c *echo.Context) error {
	setMaintenanceResponseHeaders(c)
	return apierror.ErrMaintenanceMode
}

var _ APITokenValidator = (usecase.APITokenUsecase)(nil)
