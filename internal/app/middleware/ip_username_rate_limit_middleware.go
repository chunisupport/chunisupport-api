package middleware

import (
	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/labstack/echo/v5"
)

// IPAndUsernameRateLimitMiddleware はShields.ioの共有送信元IPで別ユーザーのバッジが制限を奪い合わないようにします。
func IPAndUsernameRateLimitMiddleware(config RateLimitConfig) echo.MiddlewareFunc {
	store := newFixedWindowStoreWithCleanup(config.Window)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			key := c.RealIP() + "\x00" + c.Param("username")
			allowed, _, _ := store.Allow(key, config.Requests)
			if !allowed {
				return apierror.ErrTooManyRequests
			}
			return next(c)
		}
	}
}
