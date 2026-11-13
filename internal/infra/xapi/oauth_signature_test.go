package xapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserClient_両方のシークレットでOAuth署名する(t *testing.T) {
	var request *http.Request
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request = r
		body, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(`{"data":{"id":"123","username":"chuni_support"}}`))
	}))
	t.Cleanup(server.Close)
	_, err := newUserClient("consumer-key", "consumer-secret", server.URL, server.Client()).GetAuthenticatedUser(context.Background(), "access-token", "access-secret")
	require.NoError(t, err)
	require.NotNil(t, request)
	assert.Equal(t, http.MethodGet, request.Method)
	assert.Equal(t, "/2/users/me", request.URL.Path)
	assert.Empty(t, request.URL.RawQuery)
	assert.Empty(t, body)
	header := request.Header.Get("Authorization")
	require.True(t, strings.HasPrefix(header, "OAuth "))
	params := url.Values{}
	for _, part := range strings.Split(strings.TrimPrefix(header, "OAuth "), ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		require.True(t, ok)
		decoded, err := url.QueryUnescape(strings.Trim(value, `"`))
		require.NoError(t, err)
		params.Set(key, decoded)
	}
	signature := params.Get("oauth_signature")
	require.NotEmpty(t, signature)
	require.NotEmpty(t, params.Get("oauth_nonce"))
	require.NotEmpty(t, params.Get("oauth_timestamp"))
	params.Del("oauth_signature")
	// RFC 5849の正規化を標準ライブラリだけで行い、本番の署名器は呼ばない。
	escape := func(s string) string { return strings.ReplaceAll(url.QueryEscape(s), "+", "%20") }
	pairs := make([]string, 0, len(params))
	for key, values := range params {
		for _, value := range values {
			pairs = append(pairs, escape(key)+"="+escape(value))
		}
	}
	sort.Strings(pairs)
	base := "GET&" + escape(server.URL+"/2/users/me") + "&" + escape(strings.Join(pairs, "&"))
	mac := hmac.New(sha1.New, []byte("consumer-secret&access-secret"))
	_, err = mac.Write([]byte(base))
	require.NoError(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString(mac.Sum(nil)), signature)
}
