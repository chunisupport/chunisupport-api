package xapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserClient_GetAuthenticatedUser_本人情報を署名付きで取得する(t *testing.T) {
	// Given
	var gotPath, gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"123","name":"ちゅに","username":"chuni_support"}}`))
	}))
	t.Cleanup(server.Close)
	client := newUserClient("consumer-key", "consumer-secret", server.URL, server.Client())

	// When
	got, err := client.GetAuthenticatedUser(context.Background(), "access-token", "access-secret")

	// Then
	require.NoError(t, err)
	assert.Equal(t, usecase.XAuthenticatedUser{ID: "123", Username: "chuni_support"}, got)
	assert.Equal(t, "/2/users/me", gotPath)
	assert.True(t, strings.HasPrefix(gotAuthorization, "OAuth "))
	assert.Contains(t, gotAuthorization, `oauth_consumer_key="consumer-key"`)
	assert.Contains(t, gotAuthorization, `oauth_token="access-token"`)
	assert.Contains(t, gotAuthorization, `oauth_signature_method="HMAC-SHA1"`)
}

func TestUserClient_GetAuthenticatedUser_失敗時のエラー(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantErrIs error
	}{
		{name: "401はトークン拒否として認証情報エラー", status: http.StatusUnauthorized, body: `{}`, wantErrIs: usecase.ErrInvalidXCredentials},
		{name: "429は回数制限としてX API利用不可エラー", status: http.StatusTooManyRequests, body: `{}`, wantErrIs: usecase.ErrXAPIUnavailable},
		{name: "403はアプリ設定の問題としてX API利用不可エラー", status: http.StatusForbidden, body: `{}`, wantErrIs: usecase.ErrXAPIUnavailable},
		{name: "JSONとして読めない応答はX API利用不可エラー", status: http.StatusOK, body: `not json`, wantErrIs: usecase.ErrXAPIUnavailable},
		{name: "本人情報のない成功応答はX API利用不可エラー", status: http.StatusOK, body: `{"errors":[{"title":"Not Found"}]}`, wantErrIs: usecase.ErrXAPIUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)
			client := newUserClient("consumer-key", "consumer-secret", server.URL, server.Client())

			// When
			got, err := client.GetAuthenticatedUser(context.Background(), "access-token", "access-secret")

			// Then
			assert.ErrorIs(t, err, tt.wantErrIs)
			assert.Equal(t, usecase.XAuthenticatedUser{}, got)
		})
	}
}
