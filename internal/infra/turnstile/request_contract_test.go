package turnstile

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type requestContractTransport func(*http.Request) (*http.Response, error)

func (f requestContractTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestVerifier_本番のsiteverifyへシークレットを送信する(t *testing.T) {
	for _, ip := range []string{"", "203.0.113.1"} {
		t.Run(ip, func(t *testing.T) {
			v, ok := NewVerifier(" secret ", []string{"https://app.example.com"}).(*verifier)
			require.True(t, ok)
			called := false
			v.client = &http.Client{Transport: requestContractTransport(func(r *http.Request) (*http.Response, error) {
				called = true
				assert.Equal(t, "https://challenges.cloudflare.com/turnstile/v0/siteverify", r.URL.String())
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				form, err := url.ParseQuery(string(body))
				require.NoError(t, err)
				expected := url.Values{"secret": {"secret"}, "response": {"token"}}
				if ip != "" {
					expected.Set("remoteip", ip)
				}
				assert.Equal(t, expected, form)
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"success":true,"hostname":"app.example.com","action":"login"}`)), Request: r}, nil
			})}
			require.NoError(t, v.VerifyTurnstile(context.Background(), " token ", ip, "login"))
			assert.True(t, called)
		})
	}
}
