package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadR2ConfigFromEnv(t *testing.T) {
	tests := []struct {
		name       string
		endpoint   string
		accessKey  string
		secretKey  string
		bucketName string
		want       R2Config
		wantErr    bool
	}{
		{
			name:       "Cloudflare R2の接続情報を読み込める",
			endpoint:   " https://account-id.r2.cloudflarestorage.com ",
			accessKey:  " access-key ",
			secretKey:  " secret-key ",
			bucketName: " song-snapshots ",
			want: R2Config{
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
			t.Setenv("R2_ENDPOINT_URL", tt.endpoint)
			t.Setenv("R2_ACCESS_KEY_ID", tt.accessKey)
			t.Setenv("R2_SECRET_ACCESS_KEY", tt.secretKey)
			t.Setenv("R2_BUCKET_NAME", tt.bucketName)

			// When
			got, err := LoadR2ConfigFromEnv()

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
