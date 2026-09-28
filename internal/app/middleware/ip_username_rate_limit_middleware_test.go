package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestIPAndUsernameRateLimitMiddleware(t *testing.T) {
	e := setupEchoWithErrorHandler(t)
	limit := IPAndUsernameRateLimitMiddleware(RateLimitConfig{Requests: 1, Window: time.Minute})
	handler := limit(func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

	request := func(ip, username string) int {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/badges/users/"+username+"/rating", nil)
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPathValues(echo.PathValues{{Name: "username", Value: username}})
		if err := handler(c); err != nil {
			e.HTTPErrorHandler(c, err)
		}
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, request("192.0.2.1", "alice"))
	assert.Equal(t, http.StatusTooManyRequests, request("192.0.2.1", "alice"))
	assert.Equal(t, http.StatusOK, request("192.0.2.1", "bob"))
	assert.Equal(t, http.StatusOK, request("192.0.2.2", "alice"))
}
