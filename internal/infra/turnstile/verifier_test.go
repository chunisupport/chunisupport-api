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
		name       string
		token      string
		remoteIP   string
		statusCode int
		response   siteverifyResponse
		wantErr    error
	}{
		{
			name:       "検証成功ならエラーなし",
			token:      "turnstile-token",
			remoteIP:   "203.0.113.1",
			statusCode: http.StatusOK,
			response:   siteverifyResponse{Success: true},
		},
		{
			name:       "検証失敗ならErrInvalidTurnstileToken",
			token:      "invalid-token",
			statusCode: http.StatusOK,
			response:   siteverifyResponse{Success: false, ErrorCodes: []string{"invalid-input-response"}},
			wantErr:    usecase.ErrInvalidTurnstileToken,
		},
		{
			name:       "Siteverifyが5xxならErrInternalError",
			token:      "turnstile-token",
			statusCode: http.StatusInternalServerError,
			wantErr:    usecase.ErrInternalError,
		},
		{
			name:    "空トークンならErrInvalidTurnstileToken",
			token:   "  ",
			wantErr: usecase.ErrInvalidTurnstileToken,
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

			verifier := &verifier{
				secretKey: "secret",
				endpoint:  server.URL,
				client:    server.Client(),
			}

			err := verifier.VerifyTurnstile(context.Background(), tt.token, tt.remoteIP)

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
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		require.NoError(t, json.NewEncoder(w).Encode(siteverifyResponse{
			Success:    false,
			ErrorCodes: []string{"invalid-input-response"},
		}))
	}))
	defer server.Close()

	var logOutput bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logOutput, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	verifier := &verifier{
		secretKey: "secret",
		endpoint:  server.URL,
		client:    server.Client(),
	}

	// When
	err := verifier.VerifyTurnstile(context.Background(), "invalid-token", "")

	// Then
	assert.ErrorIs(t, err, usecase.ErrInvalidTurnstileToken)
	assert.Empty(t, logOutput.String())
}
