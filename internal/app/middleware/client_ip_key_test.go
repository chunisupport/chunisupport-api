package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestClientIPKey(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		expected   string
	}{
		{
			name:       "IPv4はアドレスのまま",
			remoteAddr: "192.0.2.1:1234",
			expected:   "192.0.2.1",
		},
		{
			name:       "IPv6は/64に丸める",
			remoteAddr: "[2001:db8:1:2:3:4:5:6]:1234",
			expected:   "2001:db8:1:2::/64",
		},
		{
			name:       "IPv4射影IPv6はIPv4として扱う",
			remoteAddr: "[::ffff:192.0.2.1]:1234",
			expected:   "192.0.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			c := e.NewContext(req, httptest.NewRecorder())

			// When
			result := ClientIPKey(c)

			// Then
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIPベースのレート制限はIPv6を64単位で数える(t *testing.T) {
	config := RateLimitConfig{Requests: 1, Window: time.Minute}
	tests := []struct {
		name       string
		middleware echo.MiddlewareFunc
	}{
		{name: "IPRateLimitMiddleware", middleware: IPRateLimitMiddleware(config)},
		{name: "AnonymousIPRateLimitMiddleware", middleware: AnonymousIPRateLimitMiddleware(config)},
		{name: "IPAndUsernameRateLimitMiddleware", middleware: IPAndUsernameRateLimitMiddleware(config)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			e := setupEchoWithErrorHandler(t)
			handler := tt.middleware(func(c *echo.Context) error { return c.NoContent(http.StatusOK) })
			request := func(remoteAddr string) int {
				t.Helper()
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = remoteAddr
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				if err := handler(c); err != nil {
					e.HTTPErrorHandler(c, err)
				}
				return rec.Code
			}

			// When / Then
			assert.Equal(t, http.StatusOK, request("[2001:db8:1:2::1]:1234"))
			assert.Equal(t, http.StatusTooManyRequests, request("[2001:db8:1:2::ffff]:1234"), "同じ/64の別アドレスは同じバケットに数える")
			assert.Equal(t, http.StatusOK, request("[2001:db8:1:3::1]:1234"), "別の/64は別のバケットに数える")
		})
	}
}
