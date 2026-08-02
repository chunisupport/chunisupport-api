package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadObjectStorageConfigFromEnv(t *testing.T) {
	tests := []struct {
		name       string
		endpoint   string
		accessKey  string
		secretKey  string
		bucketName string
		want       ObjectStorageConfig
		wantErr    bool
	}{
		{
			name:       "オブジェクトストレージの接続情報を読み込める",
			endpoint:   " https://account-id.r2.cloudflarestorage.com ",
			accessKey:  " access-key ",
			secretKey:  " secret-key ",
			bucketName: " song-snapshots ",
			want: ObjectStorageConfig{
				Endpoint:        "account-id.r2.cloudflarestorage.com",
				AccessKeyID:     "access-key",
				SecretAccessKey: "secret-key",
				BucketName:      "song-snapshots",
				Secure:          true,
			},
		},
		{
			name:       "HTTPエンドポイントは拒否する",
			endpoint:   "http://account-id.r2.cloudflarestorage.com",
			accessKey:  "access-key",
			secretKey:  "secret-key",
			bucketName: "song-snapshots",
			wantErr:    true,
		},
		{
			name:       "パス付きエンドポイントは拒否する",
			endpoint:   "https://account-id.r2.cloudflarestorage.com/path",
			accessKey:  "access-key",
			secretKey:  "secret-key",
			bucketName: "song-snapshots",
			wantErr:    true,
		},
		{
			name:       "必須値が不足していればまとめてエラーにする",
			endpoint:   "",
			accessKey:  "",
			secretKey:  "",
			bucketName: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			t.Setenv("OBJECT_STORAGE_ENDPOINT_URL", tt.endpoint)
			t.Setenv("OBJECT_STORAGE_ACCESS_KEY_ID", tt.accessKey)
			t.Setenv("OBJECT_STORAGE_SECRET_ACCESS_KEY", tt.secretKey)
			t.Setenv("OBJECT_STORAGE_BUCKET_NAME", tt.bucketName)

			// When
			got, err := LoadObjectStorageConfigFromEnv()

			// Then
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
