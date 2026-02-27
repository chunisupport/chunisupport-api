package api_internal

import (
	"net/http"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/app/apierror"
	"github.com/chunisupport/chunisupport-api/internal/domain/entity"
	"github.com/labstack/echo/v4"
)

const authCookieName = "token"

func newAuthCookie(cookieSecure bool, cookieSameSite http.SameSite, value string, maxAge int) *http.Cookie {
	cookie := &http.Cookie{
		Name:     authCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   cookieSecure,
		SameSite: cookieSameSite,
		MaxAge:   maxAge,
	}
	if maxAge < 0 {
		cookie.Expires = time.Unix(0, 0).UTC()
	}
	return cookie
}

func getUserEntityFromContext(c echo.Context) (*entity.User, error) {
	user, ok := c.Get("userEntity").(*entity.User)
	if !ok {
		return nil, apierror.ErrUnauthorized
	}
	return user, nil
}
