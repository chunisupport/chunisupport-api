package app

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/chunisupport/chunisupport-api/internal/config"
	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/labstack/echo/v5"
	echoMiddleware "github.com/labstack/echo/v5/middleware"
)

func newDefaultCORSConfig(cfg config.Config) echoMiddleware.CORSConfig {
	return newCORSConfig(cfg.CORS.AllowOrigins, cfg, func(c *echo.Context) bool {
		return isExternalCORSPath(c.Request().URL.Path)
	})
}

func newExternalCORSConfig(cfg config.Config) echoMiddleware.CORSConfig {
	allowOrigins := slices.Clone(cfg.CORS.AllowOrigins)
	if !slices.Contains(allowOrigins, info.ExternalCORSAllowOrigin) {
		allowOrigins = append(allowOrigins, info.ExternalCORSAllowOrigin)
	}

	return newCORSConfig(allowOrigins, cfg, nil)
}

func newTemporaryPlayerDataCORSConfig(cfg config.Config) echoMiddleware.CORSConfig {
	corsConfig := newExternalCORSConfig(cfg)
	corsConfig.Skipper = func(c *echo.Context) bool {
		return c.Request().URL.Path != "/internal/player-data/temp"
	}
	return corsConfig
}

func isExternalCORSPath(path string) bool {
	return path == "/healthz" || path == "/internal/player-data/temp" || path == "/internal/system/status"
}

func newCORSConfig(allowOrigins []string, cfg config.Config, skipper echoMiddleware.Skipper) echoMiddleware.CORSConfig {
	corsConfig := echoMiddleware.CORSConfig{
		Skipper:          skipper,
		AllowOrigins:     allowOrigins,
		AllowCredentials: cfg.CORS.AllowCredentials,
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderContentEncoding,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
			echo.HeaderXRequestID,
			"X-Reauth-Token",
		},
		ExposeHeaders: []string{
			echo.HeaderContentLength,
			echo.HeaderRetryAfter,
			echo.HeaderXRequestID,
		},
		MaxAge: cfg.CORS.MaxAge,
	}

	if hasWildcardCORSOrigin(allowOrigins) {
		corsConfig.UnsafeAllowOriginFunc = newCORSAllowOriginFunc(allowOrigins, cfg.CORS.AllowCredentials)
	}

	return corsConfig
}

// hasWildcardCORSOrigin は Echo v5 の AllowOrigins だけでは扱えないパターン指定があるかを判定します。
func hasWildcardCORSOrigin(allowOrigins []string) bool {
	return slices.ContainsFunc(allowOrigins, func(origin string) bool {
		return origin != "*" && strings.ContainsAny(origin, "*?")
	})
}

// newCORSAllowOriginFunc は設定ファイルの allow_origins 形式を維持したまま、Echo v4 互換のワイルドカード判定を行います。
// UnsafeAllowOriginFunc を設定すると Echo v5 は AllowOrigins を無視するため、完全一致オリジンもこの関数内で判定します。
func newCORSAllowOriginFunc(allowOrigins []string, allowCredentials bool) func(c *echo.Context, origin string) (string, bool, error) {
	type originPattern struct {
		matcher *regexp.Regexp
	}

	var exactOrigins []string
	var patterns []originPattern
	allowAll := false

	for _, allowedOrigin := range allowOrigins {
		if allowedOrigin == "*" {
			if allowCredentials {
				panic("* as allowed origin and AllowCredentials=true is insecure and not allowed")
			}
			allowAll = true
			continue
		}

		if strings.ContainsAny(allowedOrigin, "*?") {
			patterns = append(patterns, originPattern{
				matcher: mustCompileCORSOriginPattern(allowedOrigin),
			})
			continue
		}

		if _, err := parseCORSOrigin(allowedOrigin); err != nil {
			panic(fmt.Sprintf("invalid CORS allow origin %q: %v", allowedOrigin, err))
		}
		exactOrigins = append(exactOrigins, allowedOrigin)
	}

	return func(c *echo.Context, origin string) (string, bool, error) {
		if allowAll {
			return "*", true, nil
		}

		for _, allowedOrigin := range exactOrigins {
			if strings.EqualFold(allowedOrigin, origin) {
				return allowedOrigin, true, nil
			}
		}

		for _, pattern := range patterns {
			if pattern.matcher.MatchString(origin) {
				return origin, true, nil
			}
		}

		return "", false, nil
	}
}

// mustCompileCORSOriginPattern は CORS オリジンの `*` と `?` を正規表現へ変換します。
// 起動時設定の誤りはリクエスト時まで遅延させず、Echo の CORS 設定と同じく起動時に panic させます。
func mustCompileCORSOriginPattern(pattern string) *regexp.Regexp {
	if _, err := parseCORSOrigin(pattern); err != nil {
		panic(fmt.Sprintf("invalid CORS allow origin pattern %q: %v", pattern, err))
	}

	quotedPattern := regexp.QuoteMeta(pattern)
	quotedPattern = strings.ReplaceAll(quotedPattern, `\*`, `.*`)
	quotedPattern = strings.ReplaceAll(quotedPattern, `\?`, `.`)

	matcher, err := regexp.Compile(`(?i)^` + quotedPattern + `$`)
	if err != nil {
		panic(fmt.Sprintf("invalid CORS allow origin pattern %q: %v", pattern, err))
	}

	return matcher
}

// parseCORSOrigin は CORS の Origin として扱える scheme://host[:port] だけを許可します。
func parseCORSOrigin(origin string) (*url.URL, error) {
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return nil, err
	}
	if parsedOrigin.Scheme == "" || parsedOrigin.Host == "" {
		return nil, fmt.Errorf("scheme or host is missing")
	}
	if parsedOrigin.Path != "" || parsedOrigin.RawQuery != "" || parsedOrigin.Fragment != "" {
		return nil, fmt.Errorf("path, query, and fragment are not allowed")
	}

	return parsedOrigin, nil
}
