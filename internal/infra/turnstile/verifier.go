package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
)

const (
	siteverifyEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	requestTimeout     = 5 * time.Second
)

type verifier struct {
	secretKey string
	endpoint  string
	client    *http.Client
	hosts     hostnameMatcher
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	Hostname   string   `json:"hostname"`
	Action     string   `json:"action"`
	ErrorCodes []string `json:"error-codes"`
}

type hostnameMatcher struct {
	exact    map[string]struct{}
	patterns []*regexp.Regexp
}

// NewVerifier はCloudflare Turnstileの検証器を生成します。
// allowedOrigins はフロントのCORS許可オリジンです。siteverify の hostname はスキームとポートを含まないため、
// オリジンからホスト名だけを取り出して照合します。空や解釈できないオリジン、全許可のオリジンは許可ホストに含めません。
// 許可ホストが残らない設定ではログインと新規登録が常に失敗するため、起動時に警告します。
func NewVerifier(secretKey string, allowedOrigins []string) usecase.TurnstileVerifier {
	return &verifier{
		secretKey: strings.TrimSpace(secretKey),
		endpoint:  siteverifyEndpoint,
		client: &http.Client{
			Timeout: requestTimeout,
		},
		hosts: newHostnameMatcher(allowedOrigins),
	}
}

func (v *verifier) VerifyTurnstile(ctx context.Context, token string, remoteIP string, expectedAction string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return usecase.ErrInvalidTurnstileToken
	}

	form := url.Values{}
	form.Set("secret", v.secretKey)
	form.Set("response", token)
	if remoteIP = strings.TrimSpace(remoteIP); remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return errors.Join(usecase.ErrInternalError, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return errors.Join(usecase.ErrInternalError, err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		if err := resp.Body.Close(); err != nil {
			slog.Warn("failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return errors.Join(usecase.ErrInternalError, fmt.Errorf("turnstile siteverify returned status %d", resp.StatusCode))
	}

	var result siteverifyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*10)).Decode(&result); err != nil {
		return errors.Join(usecase.ErrInternalError, err)
	}
	if !result.Success {
		return usecase.ErrInvalidTurnstileToken
	}
	// 別ホストで解かれたトークンの流用を防ぐ。許可リストが空のときも拒否する。
	if !v.hosts.matches(result.Hostname) {
		return usecase.ErrInvalidTurnstileToken
	}
	// 別フォームのトークン流用を防ぐ。ウィジェットが action を付けていない場合も拒否する。
	if !actionMatches(result.Action, expectedAction) {
		return usecase.ErrInvalidTurnstileToken
	}

	return nil
}

func actionMatches(actual, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return false
	}

	return strings.TrimSpace(actual) == expected
}

func newHostnameMatcher(allowedOrigins []string) hostnameMatcher {
	exact := make(map[string]struct{})
	var patterns []*regexp.Regexp
	for _, origin := range allowedOrigins {
		host, ok := hostnameFromOrigin(origin)
		if !ok {
			continue
		}
		if strings.Contains(host, "*") {
			patterns = append(patterns, compileHostnamePattern(host))
			continue
		}
		exact[host] = struct{}{}
	}

	if len(exact) == 0 && len(patterns) == 0 {
		slog.Warn("no Turnstile allowed hosts; login and signup will reject every Turnstile token", "allowed_origins", allowedOrigins)
	}

	return hostnameMatcher{exact: exact, patterns: patterns}
}

// hostnameFromOrigin は CORS オリジンから照合用ホスト名を取り出します。
// `*` や `https://*` のような全許可や、スキームを持たない値はフロントのホストとして扱わないため除外します。
// 全許可をホスト照合に持ち込むと、別ホストで取得したトークンの流用を防げなくなるためです。
func hostnameFromOrigin(origin string) (string, bool) {
	origin = strings.TrimSpace(origin)
	if origin == "" || origin == "*" {
		return "", false
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" || host == "*" {
		return "", false
	}

	return host, true
}

func compileHostnamePattern(host string) *regexp.Regexp {
	quoted := regexp.QuoteMeta(host)
	quoted = strings.ReplaceAll(quoted, `\*`, `.*`)

	matcher, err := regexp.Compile(`(?i)^` + quoted + `$`)
	if err != nil {
		panic(fmt.Sprintf("invalid turnstile hostname pattern %q: %v", host, err))
	}

	return matcher
}

func (m hostnameMatcher) matches(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "" {
		return false
	}
	if m.contains(hostname) {
		return true
	}

	// siteverify がポート付きホストを返した場合も、オリジンから取り出したホスト名と照合する。
	host, _, err := net.SplitHostPort(hostname)
	if err != nil {
		return false
	}

	return m.contains(strings.ToLower(host))
}

func (m hostnameMatcher) contains(hostname string) bool {
	if _, ok := m.exact[hostname]; ok {
		return true
	}
	for _, pattern := range m.patterns {
		if pattern.MatchString(hostname) {
			return true
		}
	}

	return false
}

var _ usecase.TurnstileVerifier = (*verifier)(nil)
