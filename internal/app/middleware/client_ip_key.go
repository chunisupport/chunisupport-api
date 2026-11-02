package middleware

import (
	"net/netip"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
)

// ClientIPKey は送信元単位の制限に使うキーを返します。
// 一般的なIPv6回線は1利用者に /64 以上を割り当てており、アドレスのままではアドレスを変えるだけで制限を回避できるため、
// IPv6 は /64 に丸めて1つの送信元として数えます。
func ClientIPKey(c *echo.Context) string {
	ip := c.RealIP()
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ip
	}

	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}

	prefix, err := addr.Prefix(info.ClientIPv6PrefixBits)
	if err != nil {
		return ip
	}
	return prefix.String()
}
