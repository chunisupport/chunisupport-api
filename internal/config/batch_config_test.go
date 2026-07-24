package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chunisupport/chunisupport-api/internal/info"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBatchConfig_HTTPサーバー固有設定を要求しない(t *testing.T) {
	// Given
	originalWorkingDirectory, err := os.Getwd()
	require.NoError(t, err)
	temporaryWorkingDirectory := t.TempDir()
	require.NoError(t, os.Chdir(temporaryWorkingDirectory))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalWorkingDirectory))
	})

	require.NoError(t, os.MkdirAll(info.ConfigDir, 0o750))
	settings := []byte(`{
		"app_port": 3002,
		"timezone": "Asia/Tokyo",
		"logging": {
			"level": "info",
			"stdout": true
		},
		"shutdown_timeout_seconds": 20,
		"cors": {
			"allow_origins": [],
			"allow_credentials": false,
			"max_age": 3600
		},
		"database": {
			"startup": {
				"max_wait_sec": 0,
				"interval_sec": 1
			},
			"pool": {
				"max_open_conns": 2,
				"max_idle_conns": 2,
				"conn_max_lifetime_sec": 300,
				"conn_max_idle_time_sec": 60
			}
		}
	}`)
	require.NoError(t, os.WriteFile(filepath.Join(info.ConfigDir, "test.settings.json"), settings, 0o600))

	t.Setenv("APP_ENV", "test")
	t.Setenv("DB_NAME", "chunisupport")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "3306")
	t.Setenv("DB_USER", "batch")
	t.Setenv("DB_PASS", "password")
	t.Setenv("FIREBASE_CREDENTIALS_FILE", "")
	t.Setenv("TURNSTILE_SECRET_KEY", "")

	// When
	cfg, err := LoadBatchConfig()

	// Then
	require.NoError(t, err)
	assert.Equal(t, "chunisupport", cfg.Database.DbConfig.DbName)
	assert.Empty(t, cfg.Firebase.CredentialsFile)
	assert.Empty(t, cfg.Turnstile.SecretKey)
}
