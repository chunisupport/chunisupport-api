package turnstile

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifier_VerifyTurnstile(t *testing.T) {
	tests := []struct {
		name           string
		token          string
		remoteIP       string
		expectedAction string
		allowedOrigins []string
		statusCode     int
		response       siteverifyResponse
		wantErr        error
	}{
		{
			name:           "検証成功ならエラーなし",
			token:          "turnstile-token",
			remoteIP:       "203.0.113.1",
			expectedAction: "login",
			allowedOrigins: []string{"https://app.example.com"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "app.example.com", Action: "login"},
		},
		{
			name:           "検証失敗ならErrInvalidTurnstileToken",
			token:          "invalid-token",
			expectedAction: "login",
			allowedOrigins: []string{"https://app.example.com"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: false, ErrorCodes: []string{"invalid-input-response"}},
			wantErr:        usecase.ErrInvalidTurnstileToken,
		},
		{
			name:           "Siteverifyが5xxならErrInternalError",
			token:          "turnstile-token",
			expectedAction: "login",
			allowedOrigins: []string{"https://app.example.com"},
			statusCode:     http.StatusInternalServerError,
			wantErr:        usecase.ErrInternalError,
		},
		{
			name:           "空トークンならErrInvalidTurnstileToken",
			token:          "  ",
			expectedAction: "login",
			allowedOrigins: []string{"https://app.example.com"},
			wantErr:        usecase.ErrInvalidTurnstileToken,
		},
		{
			name:           "ホスト名が許可オリジンと異なるならErrInvalidTurnstileToken",
			token:          "turnstile-token",
			expectedAction: "login",
			allowedOrigins: []string{"https://app.example.com"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "evil.example", Action: "login"},
			wantErr:        usecase.ErrInvalidTurnstileToken,
		},
		{
			name:           "ホスト名が空ならErrInvalidTurnstileToken",
			token:          "turnstile-token",
			expectedAction: "login",
			allowedOrigins: []string{"https://app.example.com"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Action: "login"},
			wantErr:        usecase.ErrInvalidTurnstileToken,
		},
		{
			name:           "許可ホストが空なら成功応答でもErrInvalidTurnstileToken",
			token:          "turnstile-token",
			expectedAction: "login",
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "app.example.com", Action: "login"},
			wantErr:        usecase.ErrInvalidTurnstileToken,
		},
		{
			name:           "actionが期待値と異なるならErrInvalidTurnstileToken",
			token:          "turnstile-token",
			expectedAction: "login",
			allowedOrigins: []string{"https://app.example.com"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "app.example.com", Action: "signup"},
			wantErr:        usecase.ErrInvalidTurnstileToken,
		},
		{
			name:           "actionが空ならErrInvalidTurnstileToken",
			token:          "turnstile-token",
			expectedAction: "signup",
			allowedOrigins: []string{"https://app.example.com"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "app.example.com"},
			wantErr:        usecase.ErrInvalidTurnstileToken,
		},
		{
			name:           "期待actionが空ならErrInvalidTurnstileToken",
			token:          "turnstile-token",
			expectedAction: "  ",
			allowedOrigins: []string{"https://app.example.com"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "app.example.com", Action: "login"},
			wantErr:        usecase.ErrInvalidTurnstileToken,
		},
		{
			name:           "ポート付きオリジンはホスト名だけで照合する",
			token:          "turnstile-token",
			expectedAction: "signup",
			allowedOrigins: []string{"http://localhost:3000"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "localhost", Action: "signup"},
		},
		{
			name:           "ポート付きホスト名も許可ホストと照合する",
			token:          "turnstile-token",
			expectedAction: "login",
			allowedOrigins: []string{"http://localhost:3000"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "localhost:3000", Action: "login"},
		},
		{
			name:           "ホスト名の大文字小文字は区別しない",
			token:          "turnstile-token",
			expectedAction: "login",
			allowedOrigins: []string{"https://APP.Example.COM"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "app.example.com", Action: "login"},
		},
		{
			name:           "複数の許可オリジンのいずれかと一致すれば成功する",
			token:          "turnstile-token",
			expectedAction: "login",
			allowedOrigins: []string{"http://localhost:3000", "https://app.example.com"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "localhost", Action: "login"},
		},
		{
			name:           "ワイルドカードの許可オリジンに一致すれば成功する",
			token:          "turnstile-token",
			expectedAction: "login",
			allowedOrigins: []string{"https://*.example.pages.dev"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "preview.example.pages.dev", Action: "login"},
		},
		{
			name:           "ワイルドカードに一致しないホストは拒否する",
			token:          "turnstile-token",
			expectedAction: "login",
			allowedOrigins: []string{"https://*.example.pages.dev"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "example.pages.dev", Action: "login"},
			wantErr:        usecase.ErrInvalidTurnstileToken,
		},
		{
			name:           "全許可のCORSオリジンはTurnstileのホストとして扱わない",
			token:          "turnstile-token",
			expectedAction: "login",
			allowedOrigins: []string{"*"},
			statusCode:     http.StatusOK,
			response:       siteverifyResponse{Success: true, Hostname: "app.example.com", Action: "login"},
			wantErr:        usecase.ErrInvalidTurnstileToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedToken string
			var receivedRemoteIP string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.NoError(t, r.ParseForm())
				receivedToken = r.Form.Get("response")
				receivedRemoteIP = r.Form.Get("remoteip")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					require.NoError(t, json.NewEncoder(w).Encode(tt.response))
				}
			}))
			defer server.Close()

			verifier := newTestVerifier(t, tt.allowedOrigins, server.URL, server.Client())

			err := verifier.VerifyTurnstile(context.Background(), tt.token, tt.remoteIP, tt.expectedAction)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.token, receivedToken)
			assert.Equal(t, tt.remoteIP, receivedRemoteIP)
		})
	}
}

func TestVerifier_VerifyTurnstile_検証失敗をログに出力しない(t *testing.T) {
	// 前提
	responseBody, err := json.Marshal(siteverifyResponse{
		Success:    false,
		ErrorCodes: []string{"invalid-input-response"},
	})
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(responseBody)
	}))
	defer server.Close()

	var logOutput bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logOutput, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	verifier := newTestVerifier(t, []string{"https://app.example.com"}, server.URL, server.Client())

	// 実行
	err = verifier.VerifyTurnstile(context.Background(), "invalid-token", "", "login")

	// 検証
	assert.ErrorIs(t, err, usecase.ErrInvalidTurnstileToken)
	assert.Empty(t, logOutput.String())
}

func newTestVerifier(t *testing.T, allowedOrigins []string, endpoint string, client *http.Client) *verifier {
	t.Helper()

	verifier, ok := NewVerifier("secret", allowedOrigins).(*verifier)
	require.True(t, ok)
	verifier.endpoint = endpoint
	verifier.client = client

	return verifier
}
