package app

import (
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/chunisupport/chunisupport-api/internal/app/middleware"
	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

// newEcho はEchoインスタンスを生成し、全リクエストに共通するHTTP設定とグローバルミドルウェアを適用します。
// グローバルミドルウェアの順序は応答ヘッダーやログの内容に影響するため、この関数だけで管理します。
// echoLogWriterがnilの場合は、テストなどの直接構築時にアクセスログミドルウェアを無効化します。
func newEcho(cfg config.Config, echoLogWriter io.Writer) (*echo.Echo, error) {
	e := echo.New()
	if err := configureIPExtractor(e, cfg.ClientIP); err != nil {
		return nil, fmt.Errorf("failed to configure client IP extractor: %w", err)
	}
	e.Validator = NewCustomValidator()
	e.JSONSerializer = NewTimezoneJSONSerializer(cfg.Location)

	e.HTTPErrorHandler = middleware.CustomHTTPErrorHandler

	// アクセスログは処理後にリクエストヘッダーの X-Request-ID を読むため、先に受理または生成しておく。
	e.Use(middleware.RequestIDMiddleware())
	// エラー応答やメンテナンス応答にも付与するため、応答を返しうる後続のミドルウェアより前に置く。
	e.Use(newSecurityHeadersMiddleware())

	if echoLogWriter != nil {
		e.Logger = slog.New(slog.NewTextHandler(echoLogWriter, nil))
		e.Use(echoMiddleware.RequestLogger())
	}

	e.Use(echoMiddleware.Recover())
	e.Use(newRequestBodyLimitMiddleware())

	e.Use(echoMiddleware.CORSWithConfig(newDefaultCORSConfig(cfg)))

	return e, nil
}

func configureIPExtractor(e *echo.Echo, cfg config.ClientIP) error {
	if len(cfg.TrustedProxyCIDRs) == 0 {
		e.IPExtractor = echo.ExtractIPDirect()
		return nil
	}

	trustOptions := []echo.TrustOption{
		echo.TrustLoopback(false),
		echo.TrustLinkLocal(false),
		echo.TrustPrivateNet(false),
	}
	for _, cidr := range cfg.TrustedProxyCIDRs {
		_, trustedRange, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid trusted proxy CIDR %q: %w", cidr, err)
		}
		trustOptions = append(trustOptions, echo.TrustIPRange(trustedRange))
	}
	e.IPExtractor = echo.ExtractIPFromXFFHeader(trustOptions...)
	return nil
}

func newRequestBodyLimitMiddleware() echo.MiddlewareFunc {
	standard := echoMiddleware.BodyLimit(info.RequestBodyLimit)
	dataTransfer := echoMiddleware.BodyLimit(info.DataTransferEnvelopeMaxBytes)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		standardHandler := standard(next)
		dataTransferHandler := dataTransfer(next)
		return func(context *echo.Context) error {
			switch context.Request().URL.Path {
			case "/internal/me/data-transfer/validate", "/internal/me/data-transfer/import":
				return dataTransferHandler(context)
			default:
				return standardHandler(context)
			}
		}
	}
}
