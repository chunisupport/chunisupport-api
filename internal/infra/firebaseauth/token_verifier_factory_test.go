package firebaseauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuthClientFromCredentialsFile_起動のキャンセル後も認証リクエストを送れる(t *testing.T) {
	// Given
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	t.Setenv("FIREBASE_CONFIG", "")
	t.Setenv("GOOGLE_CLOUD_PROJECT", "")
	t.Setenv("GCLOUD_PROJECT", "")

	var tokenRequested atomic.Bool
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		tokenRequested.Store(true)
		// OAuth トークンを発行せず、実際の Firebase API へのアクセスを防ぎます。
		http.Error(w, "test token exchange rejected", http.StatusBadRequest)
	}))
	defer tokenServer.Close()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	credentials, err := json.Marshal(map[string]string{
		"type":         "service_account",
		"project_id":   "test-project",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"private_key":  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})),
		"token_uri":    tokenServer.URL,
	})
	require.NoError(t, err)
	credentialsFile := filepath.Join(t.TempDir(), "firebase.json")
	require.NoError(t, os.WriteFile(credentialsFile, credentials, 0o600))

	startupCtx, cancelStartup := context.WithCancel(t.Context())
	defer cancelStartup()
	client, err := NewAuthClientFromCredentialsFile(startupCtx, credentialsFile)
	require.NoError(t, err)
	cancelStartup()

	// When
	requestCtx, cancelRequest := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancelRequest()
	_, err = client.GetUser(requestCtx, "test-user")

	// Then
	assert.Error(t, err)
	assert.True(t, tokenRequested.Load(), "起動時のキャンセル後に OAuth トークンの取得を試みる")
}
