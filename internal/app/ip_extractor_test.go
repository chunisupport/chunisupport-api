package app

import (
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureIPExtractor_直接接続ではRemoteAddrを使用する(t *testing.T) {
	e := echo.New()
	require.NoError(t, configureIPExtractor(e, config.ClientIP{}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "198.51.100.10:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.10")

	assert.Equal(t, "198.51.100.10", e.IPExtractor(req))
}

func TestConfigureIPExtractor_信頼したプロキシのXFFからクライアントIPを取得する(t *testing.T) {
	e := echo.New()
	require.NoError(t, configureIPExtractor(e, config.ClientIP{
		TrustedProxyCIDRs: []string{"10.0.0.0/8"},
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.1.2.3:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.10, 10.1.2.4")

	assert.Equal(t, "198.51.100.10", e.IPExtractor(req))
}

func TestConfigureIPExtractor_不正なプロキシCIDRを拒否する(t *testing.T) {
	e := echo.New()

	err := configureIPExtractor(e, config.ClientIP{
		TrustedProxyCIDRs: []string{"not-a-cidr"},
	})

	assert.Error(t, err)
}
